// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/pages"
)

//go:embed content
var content embed.FS

func main() {
	b := core.NewBody("Pages Example")
	pg := pages.NewPage(b).SetSource(fsx.Sub(content, "content"))
	b.AddAppBar(pg.MakeToolbar)
	b.RunMainWindow()
}
