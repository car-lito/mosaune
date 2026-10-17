package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/tetratelabs/wazero/api"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/services"
	"slices"
)

// Service is trusted host code. It must validate operations/parameters, bound
// IO, obey ctx, sanitize replies and keep all credentials outside guest memory.
type Service func(context.Context, string, json.RawMessage) (json.RawMessage, error)
type ServiceGrants map[string]Service

var errServiceOperation = errors.New("undeclared service operation")

type serviceContextKey struct{}
type serviceSession struct {
	grants    ServiceGrants
	remaining int
}

// Intersect explicit host grants with declared operations. A declaration alone
// cannot create a grant; an undeclared operation cannot reach the host callback.
func restrictServices(ctx context.Context, declared []protocol.ServiceRequirement) context.Context {
	original, _ := ctx.Value(serviceContextKey{}).(*serviceSession)
	if original == nil {
		return ctx
	}
	allowed := ServiceGrants{}
	for _, declaration := range declared {
		service := original.grants[declaration.Name]
		if service == nil {
			continue
		}
		operations := slices.Clone(declaration.Operations)
		allowed[declaration.Name] = func(call context.Context, op string, raw json.RawMessage) (json.RawMessage, error) {
			if !slices.Contains(operations, op) {
				return nil, errServiceOperation
			}
			return service(call, op, raw)
		}
	}
	return context.WithValue(ctx, serviceContextKey{}, &serviceSession{grants: allowed, remaining: 4})
}

func (e *Engine) installServices(ctx context.Context) error {
	_, err := e.vm.NewHostModuleBuilder(services.Module).NewFunctionBuilder().WithFunc(hostRead).Export("read").Instantiate(ctx)
	return err
}
func hostRead(ctx context.Context, module api.Module, input, size, output, capacity uint32) uint32 {
	session, _ := ctx.Value(serviceContextKey{}).(*serviceSession)
	if session == nil || session.remaining <= 0 || size > services.MaxRequest || size == 0 || capacity > services.MaxResponse {
		return 0
	}
	session.remaining-- // Invalid and denied calls also consume the budget.
	memory := module.Memory()
	if memory == nil {
		return 0
	}
	data, ok := memory.Read(input, size)
	if !ok {
		return 0
	}
	if _, ok = memory.Read(output, capacity); !ok {
		return 0
	}
	var request services.Request
	if protocol.DecodeOne(data, &request) != nil || request.Validate() != nil {
		return 0
	}
	reply := services.Reply{Error: "service_denied"}
	if service := session.grants[request.Service]; service != nil {
		// Never expose callback error messages (which may contain URLs or secrets).
		body, err := service(ctx, request.Operation, append(json.RawMessage(nil), request.Params...))
		if err != nil {
			reply.Error = "service_failed"
		} else if len(body) > 2<<20 || !json.Valid(body) {
			reply.Error = "service_invalid_response"
		} else {
			reply = services.Reply{Data: body}
		}
	}
	if ctx.Err() != nil {
		return 0
	}
	encoded, err := json.Marshal(reply)
	if err != nil || len(encoded) > int(capacity) || !memory.Write(output, encoded) {
		return 0
	}
	return uint32(len(encoded))
}

// CallWithServices grants services only to this invocation. Neither a manifest
// nor a guest can grant itself a service. The grants map is copied on entry.
func (e *Engine) CallWithServices(ctx context.Context, id string, request protocol.Request, grants ServiceGrants) (Result, error) {
	copyGrants := make(ServiceGrants, len(grants))
	for name, service := range grants {
		copyGrants[name] = service
	}
	return e.Call(context.WithValue(ctx, serviceContextKey{}, &serviceSession{grants: copyGrants, remaining: 4}), id, request)
}
