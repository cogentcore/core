# Getting started with Pages

To get started with pages, make a new Go file with this code:

```go
package main

import (
	"embed"
	"io/fs"

	"cogentcore.org/core/core"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/pages"
)

//go:embed content
var content embed.FS

func main() {
	b := core.NewBody("Pages Example")
	pg := pages.NewPage(b).SetSource(errors.Log1(fs.Sub(content, "content")))
	b.AddAppBar(pg.AppBar)
	pg.OpenURL("", true)
	b.RunMainWindow()
}
```
