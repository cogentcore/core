// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"io/fs"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
)

// SetAppHelp sets the help of the app ([core.AppHelp]), from the Help item
// of the app menu, to open the given documentation of the app, whose home
// page is home.md, with [OpenHelp].
func SetAppHelp(docs fs.FS) {
	core.AppHelp = func(ctx core.Widget) {
		OpenHelp(ctx, docs)
	}
}

// OpenHelp opens the given documentation of the app in a window, in the
// context of the given widget, or raises the window if it is open already.
// The content does not use the URL of the web page, which is the app's.
func OpenHelp(ctx core.Widget, docs fs.FS) {
	if core.RecycleMainWindow(docs) {
		return
	}
	b := core.NewBody(core.TheApp.Name() + " help").SetData(docs)
	ct := NewContent(b)
	ct.noWebURL = true // before the source, which opens the first page
	ct.SetSource(docs)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			ct.MakeToolbar(p)
		})
	})
	b.NewWindow().SetContext(ctx).SetNewWindow(true).Run()
}
