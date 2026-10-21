// Package runtime is an experimental WASI command runner used for bundled
// modules. Arbitrary executable packages remain excluded from the installer.
package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
	"mosaune/extensionkit/dependencies"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/services"
)

var (
	ErrUnavailable = errors.New("extension unavailable or disabled")
	ErrBusy        = errors.New("extension engine busy; retry later")
	ErrLimit       = errors.New("extension resource limit exceeded")
	ErrProtocol    = errors.New("invalid extension response")
	ErrExecution   = errors.New("extension execution failed")
)

// Limits are set by the host, never accepted from a guest. Linear memory is not
// a total process-RAM budget: compilation, stacks and engine metadata add costs.
type Limits struct {
	// Interpreter is a trusted host choice for short-lived, small workloads.
	// It avoids native compilation at startup; guest manifests cannot select it.
	Interpreter                                           bool
	MemoryMiB                                             uint32
	Timeout                                               time.Duration
	CompileTimeout                                        time.Duration
	CodeBytes, InputBytes, OutputBytes, LogBytes, Modules int
}

func DefaultLimits() Limits {
	return Limits{MemoryMiB: 64, Timeout: 2 * time.Second, CompileTimeout: 15 * time.Second, CodeBytes: 8 << 20, InputBytes: 64 << 10, OutputBytes: 256 << 10, LogBytes: 8 << 10, Modules: 4}
}
func (l Limits) validate() error {
	if l.CompileTimeout < time.Second || l.CompileTimeout > 30*time.Second {
		return errors.New("invalid compilation timeout")
	}
	if l.MemoryMiB < 16 || l.MemoryMiB > 256 || l.Timeout < 10*time.Millisecond || l.Timeout > 30*time.Second || l.CodeBytes < 8 || l.CodeBytes > 16<<20 || l.InputBytes < 128 || l.InputBytes > 64<<10 || l.OutputBytes < 128 || l.OutputBytes > 1<<20 || l.LogBytes < 1 || l.LogBytes > 64<<10 || l.Modules < 1 || l.Modules > 16 {
		return errors.New("invalid host limits")
	}
	return nil
}

type plugin struct {
	descriptor protocol.Descriptor
	bindings   map[string]string
	compiled   wazero.CompiledModule
	enabled    bool
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
	calls      sync.WaitGroup
}
type Engine struct {
	mu      sync.Mutex
	vm      wazero.Runtime
	limits  Limits
	plugins map[string]*plugin
	gate    chan struct{}
	closed  bool
	ctx     context.Context
	cancel  context.CancelFunc
	jobs    sync.WaitGroup
}
type Metrics struct {
	ElapsedMS   float64 `json:"elapsedMs"`
	MemoryBytes uint64  `json:"linearMemoryBytes"`
	OutputBytes int     `json:"outputBytes"`
}
type Result struct {
	Response protocol.Response `json:"response"`
	Metrics  Metrics           `json:"metrics"`
}

func New(ctx context.Context, l Limits) (*Engine, error) {
	if err := l.validate(); err != nil {
		return nil, err
	}
	lifetime, cancel := context.WithCancel(ctx)
	config := wazero.NewRuntimeConfig()
	if l.Interpreter {
		config = wazero.NewRuntimeConfigInterpreter()
	}
	config = config.WithMemoryLimitPages(l.MemoryMiB * 16).WithCloseOnContextDone(true)
	vm := wazero.NewRuntimeWithConfig(lifetime, config)
	if _, err := wasi_snapshot_preview1.Instantiate(lifetime, vm); err != nil {
		cancel()
		_ = vm.Close(context.Background())
		return nil, err
	}
	e := &Engine{vm: vm, limits: l, plugins: map[string]*plugin{}, gate: make(chan struct{}, 1), ctx: lifetime, cancel: cancel}
	if err := e.installServices(lifetime); err != nil {
		cancel()
		_ = vm.Close(context.Background())
		return nil, err
	}
	return e, nil
}
func (e *Engine) acquire(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed || e.ctx.Err() != nil {
		return ErrUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case e.gate <- struct{}{}:
		e.jobs.Add(1)
		return nil
	default:
		return ErrBusy
	}
}
func (e *Engine) release() { <-e.gate; e.jobs.Done() }
func (e *Engine) invocation(ctx context.Context, timeout time.Duration) (context.Context, func()) {
	call, cancel := context.WithTimeout(ctx, timeout)
	stop := context.AfterFunc(e.ctx, cancel)
	return call, func() { stop(); cancel() }
}

// Load compiles a module but never executes it or auto-enables it. No shared
// compilation cache or filesystem is mounted. Gate bounds simultaneous jobs.
func (e *Engine) Load(ctx context.Context, d protocol.Descriptor, code []byte) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if len(code) > e.limits.CodeBytes {
		return ErrLimit
	}
	if err := e.acquire(ctx); err != nil {
		return err
	}
	defer e.release()
	e.mu.Lock()
	_, exists := e.plugins[d.ID]
	full := len(e.plugins) >= e.limits.Modules
	e.mu.Unlock()
	if exists || full {
		return errors.New("duplicate extension or module limit reached")
	}
	call, done := e.invocation(ctx, e.limits.CompileTimeout)
	defer done()
	compiled, err := e.vm.CompileModule(call, code)
	if err != nil {
		return fmt.Errorf("compile extension: %w", err)
	}
	keep := false
	defer func() {
		if !keep {
			_ = compiled.Close(context.Background())
		}
	}()
	for _, f := range compiled.ImportedFunctions() {
		module, name, _ := f.Import()
		if module == services.Module && name == "read" {
			continue
		}
		if module != wasi_snapshot_preview1.ModuleName || strings.HasPrefix(name, "sock_") {
			return errors.New("unsupported host import")
		}
	}
	if len(compiled.ImportedMemories()) != 0 {
		return errors.New("shared guest memory is not allowed")
	}
	if compiled.ExportedFunctions()["_start"] == nil {
		return errors.New("WASI command entry point _start required")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed || call.Err() != nil {
		return ErrUnavailable
	}
	// Own all slices: a caller must not change declared permissions after Load.
	d.Capabilities = slices.Clone(d.Capabilities)
	d.Dependencies = slices.Clone(d.Dependencies)
	d.Services = slices.Clone(d.Services)
	for i := range d.Services {
		d.Services[i].Operations = slices.Clone(d.Services[i].Operations)
	}
	e.plugins[d.ID] = &plugin{descriptor: d, compiled: compiled}
	keep = true
	return nil
}
func (e *Engine) Enable(id string) error {
	return e.EnableWithProviders(id, nil)
}

// EnableWithProviders binds declared capabilities to active providers. Choices
// are made by the host; changing an active binding requires disabling first.
func (e *Engine) EnableWithProviders(id string, choices map[string]string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.plugins[id]
	if e.closed || e.ctx.Err() != nil || p == nil {
		return ErrUnavailable
	}
	if p.enabled {
		for capability, selected := range choices {
			if selected == "" || p.bindings[capability] != selected {
				return errors.New("disable extension before changing its providers")
			}
		}
		return nil
	}
	available := make([]dependencies.Candidate, 0, len(e.plugins))
	for _, candidate := range e.plugins {
		available = append(available, dependencies.Candidate{Descriptor: candidate.descriptor, Enabled: candidate.enabled})
	}
	bindings, err := dependencies.Resolve(p.descriptor, available, choices)
	if err != nil {
		return err
	}
	if !p.enabled {
		p.bindings = bindings
		p.ctx, p.cancel = context.WithCancel(e.ctx)
		p.enabled = true
	}
	return nil
}

// Disable cancels the current generation, including an invocation already in
// progress. Re-enabling creates a new generation; a canceled result is discarded.
func (e *Engine) Disable(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.plugins[id]
	if p == nil {
		return ErrUnavailable
	}
	if dependents := e.dependentsLocked(id); len(dependents) > 0 {
		return &DependentsError{Module: id, Dependents: dependents}
	}
	if p.enabled {
		p.enabled = false
		p.cancel()
	}
	return nil
}
func (e *Engine) Unload(id string) error {
	e.mu.Lock()
	p := e.plugins[id]
	if p == nil {
		e.mu.Unlock()
		return ErrUnavailable
	}
	if dependents := e.dependentsLocked(id); len(dependents) > 0 {
		e.mu.Unlock()
		return &DependentsError{Module: id, Dependents: dependents}
	}
	delete(e.plugins, id)
	if p.cancel != nil {
		p.cancel()
	}
	e.mu.Unlock()
	p.calls.Wait()
	return p.compiled.Close(context.Background())
}

func (e *Engine) Call(ctx context.Context, id string, request protocol.Request) (Result, error) {
	var result Result
	if len(request.Payload) > e.limits.InputBytes {
		return result, ErrLimit
	}
	if err := request.Validate(); err != nil {
		return result, err
	}
	input, err := json.Marshal(request)
	if err != nil {
		return result, err
	}
	if len(input) > e.limits.InputBytes {
		return result, ErrLimit
	}
	if err = e.acquire(ctx); err != nil {
		return result, err
	}
	defer e.release()
	e.mu.Lock()
	p := e.plugins[id]
	if p == nil || !p.enabled || !slices.Contains(p.descriptor.Capabilities, request.Capability) {
		e.mu.Unlock()
		return result, ErrUnavailable
	}
	generation := p.ctx
	p.calls.Add(1)
	p.running = true
	e.mu.Unlock()
	defer func() { e.mu.Lock(); p.running = false; e.mu.Unlock(); p.calls.Done() }()
	call, done := e.invocation(ctx, e.limits.Timeout)
	defer done()
	call = restrictServices(call, p.descriptor.Services)
	stop := context.AfterFunc(generation, done)
	defer stop()
	if generation.Err() != nil {
		return result, ErrUnavailable
	}
	stdout := &boundedWriter{max: e.limits.OutputBytes}
	stderr := &boundedWriter{max: e.limits.LogBytes}
	// No WithFS/WithFSConfig, environment, secrets, inherited args, socket
	// descriptors. Named services are available only with per-invocation grants.
	config := wazero.NewModuleConfig().WithName("").WithStdin(bytes.NewReader(input)).WithStdout(stdout).WithStderr(stderr)
	start := time.Now()
	module, err := e.vm.InstantiateModule(call, p.compiled, config)
	result.Metrics.ElapsedMS = float64(time.Since(start).Microseconds()) / 1000
	result.Metrics.OutputBytes = stdout.data.Len()
	if module != nil {
		if memory := module.Memory(); memory != nil {
			result.Metrics.MemoryBytes = uint64(memory.Size())
		}
		_ = module.Close(context.Background())
	}
	if generation.Err() != nil {
		return result, ErrUnavailable
	}
	if call.Err() != nil {
		return result, call.Err()
	}
	if stdout.exceeded || stderr.exceeded {
		return result, ErrLimit
	}
	if err != nil {
		var exit *sys.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 0 {
			return result, ErrExecution
		}
	}
	if err = protocol.DecodeOne(stdout.data.Bytes(), &result.Response); err != nil {
		return Result{Metrics: result.Metrics}, ErrProtocol
	}
	if err = result.Response.Validate(request); err != nil {
		return Result{Metrics: result.Metrics}, ErrProtocol
	}
	return result, nil
}
func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	e.cancel()
	e.mu.Unlock()
	e.jobs.Wait()
	return e.vm.Close(context.Background())
}

type boundedWriter struct {
	data     bytes.Buffer
	max      int
	exceeded bool
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	if len(p) > w.max-w.data.Len() {
		w.exceeded = true
		return 0, io.ErrShortBuffer
	}
	return w.data.Write(p)
}
