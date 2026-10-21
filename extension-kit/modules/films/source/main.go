// Independent WASI entry point: no database, files, credentials or network.
package main

import (
	recognition "mosaune/extensionkit/capabilities/recognition/v1"
	"mosaune/extensionkit/modules/recognizer"
	"mosaune/extensionkit/modules/video"
)

func main() {
	recognizer.Serve(recognition.Formats{Extensions: []string{".mp4", ".mkv", ".webm", ".mov", ".avi", ".m4v", ".ts", ".mpeg", ".mpg"}}, func(p string) recognition.Item { return video.Identify("films", p) })
}
