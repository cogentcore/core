// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	_ "cogentcore.org/core/text/tex"
	"cogentcore.org/core/tree"
	_ "cogentcore.org/core/yaegicore"
)

//go:embed content
var econtent embed.FS

func main() {
	content.Settings.SiteTitle = "Cogent Content Example"
	content.OfflineURL = "https://example.com"
	b := core.NewBody("Cogent Content Example")
	ct := content.NewContent(b).SetContent(econtent)
	ct.Context.AddWikilinkHandler(htmlcore.GoDocWikilink("doc", "cogentcore.org/core"))
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			ct.MakeToolbar(p)
			ct.MakeToolbarPDF(p)
		})
	})
	b.RunMainWindow()
}
