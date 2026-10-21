// Independent WASI entry point: no database, files, credentials or network.
package main

import (
	recognition "mosaune/extensionkit/capabilities/recognition/v1"
	"mosaune/extensionkit/modules/recognizer"
	"path"
	"strings"
)

func main() {
	recognizer.Serve(recognition.Formats{Extensions: []string{".mp3", ".flac", ".m4a", ".aac", ".ogg", ".opus", ".wav", ".aiff", ".wma"}}, func(p string) recognition.Item {
		title := strings.TrimSuffix(path.Base(p), path.Ext(p))
		if strings.TrimSpace(title)=="" { title=path.Base(p) }
		return recognition.Item{Title: strings.Join(strings.Fields(title)," ")}
	})
}
