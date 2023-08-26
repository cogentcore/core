// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
)

// GUI starts the GUI for the given
// Gear app, which must be passed as
// a pointer.
func GUI(app any) {
	gimain.Main(func() {
		mainrun(app)
	})
}

func mainrun(app any) {
	win := gi.NewMainWindow("gear", "Gear", 1024, 768)
	vp := win.WinViewport2D()
	updt := vp.UpdateStart()
	mfr := win.SetMainFrame()

	title := gi.AddNewLabel(mfr, "title", "Gear")
	title.Type = gi.LabelHeadlineSmall

	sv := giv.AddNewStructView(mfr, "sv")
	sv.SetStruct(app)

	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}
