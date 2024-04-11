# Getting started with Webcore

To get started with Webcore, make a new Go file with this code:

```go
package main

import (
	"embed"
	"io/fs"

	"cogentcore.org/core/core"
	"cogentcore.org/core/errors"
	"cogentcore.org/core/webcore"
)

//go:embed content
var content embed.FS

func main() {
	b := core.NewBody("Webcore Example")
	pg := webcore.NewPage(b).SetSource(errors.Log1(fs.Sub(content, "content")))
	b.AddAppBar(pg.AppBar)
	pg.OpenURL("", true)
	b.RunMainWindow()
}
```
