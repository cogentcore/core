// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"
	"io/fs"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/webcore"
)

//go:embed content
var content embed.FS

func main() {
	b := gi.NewAppBody("Webcore Example")
	pg := webcore.NewPage(b).SetSource(grr.Log1(fs.Sub(content, "content")))
	b.AddAppBar(pg.AppBar)
	pg.OpenURL("", true)
	b.RunMainWindow()
}
