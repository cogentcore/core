// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/goki/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {

	tstslice := make([]string, 10)

	for i := 0; i < len(tstslice); i++ {
		tstslice[i] = fmt.Sprintf("this is element: %v", i)
	}

	tstmap := make(map[string]string)

	tstmap["mapkey1"] = "whatever"
	tstmap["mapkey2"] = "testing"
	tstmap["mapkey3"] = "boring"

	// turn this on to see a trace of the rendering
	// gi.Render2DTrace = true
	// gi.Layout2DTrace = true

	width := 1024
	height := 768
	win := gi.NewWindow2D("gogi-views-test", "GoGi Views Test", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()
	vp.Fill = true

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	trow := vlay.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutRow
	trow.SetProp("horizontal-align", "center")
	trow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	trow.SetStretchMaxWidth()

	spc := vlay.AddNewChild(gi.KiT_Space, "spc1").(*gi.Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	trow.AddNewChild(gi.KiT_Stretch, "str1")
	lab1 := trow.AddNewChild(gi.KiT_Label, "lab1").(*gi.Label)
	lab1.Text = "<large>This is a test of the <tt>Slice</tt> and <tt>Map</tt> Views reflect-ive GUI</large>"
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	trow.AddNewChild(gi.KiT_Stretch, "str2")

	split := vlay.AddNewChild(gi.KiT_SplitView, "split").(*gi.SplitView)
	split.Dim = gi.X

	mvfr := split.AddNewChild(gi.KiT_Frame, "mvfr").(*gi.Frame)
	svfr := split.AddNewChild(gi.KiT_Frame, "svfr").(*gi.Frame)
	split.SetSplits(.5, .5)

	mv := mvfr.AddNewChild(giv.KiT_MapView, "mv").(*giv.MapView)
	mv.SetMap(&tstmap, nil)
	mv.SetStretchMaxWidth()
	mv.SetStretchMaxHeight()

	sv := svfr.AddNewChild(giv.KiT_SliceView, "sv").(*giv.SliceView)
	sv.SetSlice(&tstslice, nil)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
