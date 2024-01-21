# Getting started with WebKi

To get started with WebKi, make a new Go file with this code:

```go
package main

import (
	"embed"
	"io/fs"

	"goki.dev/gi/v2/gi"
	"goki.dev/grr"
	"goki.dev/webki"
)

//go:embed content/*
var content embed.FS

func main() {
	sc := gi.NewScene("webki-basic")
	grr.Log0(webki.NewPage(sc).SetSource(grr.Log(fs.Sub(content, "content"))).OpenURL(""))
	gi.NewWindow(sc).Run().Wait()
}

```