// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"embed"
	"fmt"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	_ "cogentcore.org/core/text/tex"
	"cogentcore.org/core/text/tex/texcache"
)

//go:embed content
var econtent embed.FS

//go:embed mathcache.json.gz
var mathcache embed.FS

func main() {
	texcache.OpenFS(mathcache, "mathcache.json.gz") // note: doesn't work on web / js
	content.Settings.SiteTitle = "Generate Cache Math"
	content.OfflineURL = "https://example.com"
	b := core.NewBody("Generate Cache Math")
	ct := content.NewContent(b).SetContent(econtent)
	b.OnShow(func(e events.Event) {
		ct.LoadEachPage(func() {
			texcache.DeleteUnused()
			errors.Log(texcache.SaveAs("mathcache.json.gz"))
			fmt.Println("ALL DONE!")
			core.TheApp.QuitReq()
		})
	})
	b.RunMainWindow()
}
