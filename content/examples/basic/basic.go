// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"embed"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/text/tex/texcache"
	"cogentcore.org/core/tree"
	_ "cogentcore.org/core/yaegicore"
)

//go:generate go run ./genmath.go

//go:embed mathcache.json.gz
var mathcache []byte

//go:embed content
var econtent embed.FS

func main() {
	texcache.ReadGzip(bytes.NewBuffer(mathcache))
	texcache.SetShapeMath() // only use cached!
	// rasterx.UseGlyphCache = false
	content.Settings.SiteTitle = "Cogent Content Example"
	content.OfflineURL = "https://example.com"
	b := core.NewBody("Cogent Content Example")
	ct := content.NewContent(b).SetContent(econtent)
	ct.Context.AddWikilinkHandler(htmlcore.GoDocWikilink("doc", "cogentcore.org/core"))
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			ct.MakeToolbar(p)
			ct.MakeToolbarPDF(p)
			tree.Add(p, func(w *core.Button) {
				w.SetText("SaveMath").SetIcon(icons.Search).SetKey(keymap.Find).
					SetTooltip("Save cached math rendering")
				w.OnClick(func(e events.Event) {
					e.SetHandled()
					errors.Log(texcache.SaveAs("mathcache.json"))
				})
			})
		})
	})
	b.RunMainWindow()
}
