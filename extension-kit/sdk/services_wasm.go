//go:build wasip1 && wasm

package sdk

import (
	"runtime"
	"unsafe"
)

//go:wasmimport mosaune_services_v1 read
func hostRead(input uint32, size uint32, output uint32, capacity uint32) int32

func readService(input, output []byte) int32 {
	n := hostRead(uint32(uintptr(unsafe.Pointer(&input[0]))), uint32(len(input)), uint32(uintptr(unsafe.Pointer(&output[0]))), uint32(len(output)))
	runtime.KeepAlive(input)
	runtime.KeepAlive(output)
	return n
}
