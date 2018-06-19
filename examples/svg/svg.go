// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver"
	"github.com/goki/ki"
)

func main() {
	driver.Main(func(app oswin.App) {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768

	// turn this on to see a trace of the rendering
	// gi.Update2DTrace = true
	// gi.Render2DTrace = true
	// gi.Layout2DTrace = true

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	win := gi.NewWindow2D("GoGi SVG Test Window", width, height, true)
	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	vp.SetProp("background-color", "#FFF")
	vp.Fill = true

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	row1 := vlay.AddNewChild(gi.KiT_Layout, "row1").(*gi.Layout)
	row1.Lay = gi.LayoutRow
	row1.SetProp("align-vert", gi.AlignMiddle)
	row1.SetProp("align-horiz", "center")
	row1.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	row1.SetStretchMaxWidth()
	row1.SetStretchMaxHeight()

	svg := row1.AddNewChild(gi.KiT_SVG, "svg").(*gi.SVG)
	svg.Fill = true

	svg.LoadXML("/Users/oreilly/go/src/github.com/srwiley/oksvg/testdata/testIcons/astronaut.svg")

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
