//go:build !wasip1 || !wasm

package sdk

func readService(input, output []byte) int32 { return -1 }
