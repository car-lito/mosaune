// Build an unverified local module archive. This does not sign or publish it.
package main

import (
	"fmt"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"os"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Usage: package-module extension.json module.wasm output.mosaune.zip")
		os.Exit(2)
	}
	manifest, err := os.ReadFile(os.Args[1])
	check(err)
	var m packageformat.ExecutableManifest
	check(protocol.DecodeOne(manifest, &m))
	code, err := os.ReadFile(os.Args[2])
	check(err)
	raw, err := packageformat.CreateExecutable(m, code)
	check(err)
	check(os.WriteFile(os.Args[3], raw, 0600))
}
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
