// Adversarial test fixture; never offered by the application installer.
package main

import (
	"encoding/json"
	"fmt"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
	"strings"
)

var count int

func main() {
	_ = sdk.Serve(os.Stdin, os.Stdout, func(r protocol.Request) (any, *protocol.Failure) {
		switch r.Operation {
		case "service":
			var input struct {
				Service   string
				Operation string
				Params    map[string]any
				Count     int
			}
			_ = json.Unmarshal(r.Payload, &input)
			if input.Count < 1 || input.Count > 8 {
				input.Count = 1
			}
			results := []map[string]any{}
			for i := 0; i < input.Count; i++ {
				data, err := sdk.ReadService(input.Service, input.Operation, input.Params)
				if err != nil {
					results = append(results, map[string]any{"error": err.Error()})
				} else {
					results = append(results, map[string]any{"data": data})
				}
			}
			return map[string]any{"results": results}, nil
		case "loop":
			for {
			}
		case "panic":
			panic("fixture only")
		case "memory":
			b := make([]byte, 128<<20)
			for i := range b {
				b[i] = 1
			}
			return map[string]any{"bytes": len(b), "last": b[len(b)-1]}, nil
		case "output":
			fmt.Print(strings.Repeat("x", 32<<10))
			os.Exit(0)
		case "logs":
			fmt.Fprint(os.Stderr, strings.Repeat("x", 32<<10))
		case "invalid":
			fmt.Print("not json")
			os.Exit(0)
		case "extra":
			fmt.Print(`{} {}`)
			os.Exit(0)
		case "wrong_id":
			fmt.Print(`{"protocol":1,"id":"wrong","result":{}}`)
			os.Exit(0)
		case "probe":
			var p struct {
				Path string `json:"path"`
			}
			_ = json.Unmarshal(r.Payload, &p)
			_, err := os.ReadFile(p.Path)
			writeErr := os.WriteFile(p.Path, []byte("overwrite"), 0600)
			return map[string]any{"readDenied": err != nil, "writeDenied": writeErr != nil, "environmentEmpty": len(os.Environ()) == 0}, nil
		}
		count++
		return map[string]any{"count": count}, nil
	})
}
