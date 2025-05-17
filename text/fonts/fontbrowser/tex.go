// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/tex"
)

func init() {
	tex.LMFontsLoad()
}

// SelectEmbedded selects an embedded font from a list.
func (fb *Browser) SelectEmbedded() { //types:add
	d := core.NewBody("Select Font")
	d.SetTitle("Select an embedded font")
	si := 0
	fl := tex.LMFonts
	names := make([]string, len(fl))
	for i := range fl {
		names[i] = fl[i].Family
	}
	tb := core.NewList(d)
	tb.SetSlice(&names).BindSelect(&si)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddOK(bar).OnClick(func(e events.Event) {
			fb.Font = fl[si].Fonts[0]
			fb.UpdateRuneMap()
			fb.Update()
		})
	})
	d.RunWindowDialog(fb)
}
