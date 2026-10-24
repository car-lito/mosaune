// This contributor example is not linked into the server. Install the archive
// to use it. Its intentionally visible prefix proves which module did the work.
package main

import (
	recognition "mosaune/extensionkit/capabilities/recognition/v1"
	"mosaune/extensionkit/modules/recognizer"
	"path"
	"strings"
)

func main() {
	recognizer.Serve(recognition.Formats{Extensions: []string{".mkv", ".mp4"}}, func(p string) recognition.Item {
		name := strings.TrimSuffix(path.Base(p), path.Ext(p))
		if len(name) > 100 {
			name = "Film de démonstration"
		}
		return recognition.Item{Title: "Exemple — " + strings.ReplaceAll(name, ".", " ")}
	})
}
