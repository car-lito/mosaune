package main

import (
	"mosaune/extensionkit/modules/cataloguer"
	"mosaune/extensionkit/modules/cataloguer/rules"
)

func main() { cataloguer.Serve(rules.Music) }
