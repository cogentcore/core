// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
)

//go:embed content
var fsys embed.FS

func main() {
	b := core.NewBody("Cogent Content Example")
	ct := content.NewContent(b).SetContent(fsys)
	ct.Context.AddWikilinkHandler(htmlcore.GoDocWikilink("doc", "cogentcore.org/core"))
	b.RunMainWindow()
}
