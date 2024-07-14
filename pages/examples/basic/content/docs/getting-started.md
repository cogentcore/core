# Getting started with Pages

To get started with pages, make a new Go file with this code:

```go
package main

import (
	"embed"
	"io/fs"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/pages"
)

//go:embed content
var content embed.FS

func main() {
	b := core.NewBody("Pages Example")
	pg := pages.NewPage(b).SetContent(content)
	b.AddAppBar(pg.MakeToolbar)
	b.RunMainWindow()
}
```
