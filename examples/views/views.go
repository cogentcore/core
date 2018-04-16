// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/rcoreilly/goki/gi"
	"github.com/rcoreilly/goki/gi/oswin"
	_ "github.com/rcoreilly/goki/gi/oswin/init"
	"github.com/rcoreilly/goki/gi/units"
)

func main() {
	go mainrun()
	oswin.RunBackendEventLoop() // this needs to run in main loop
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

	width := 800
	height := 800
	win := gi.NewWindow2D("Views Window", width, height)
	win.UpdateStart()

	vp := win.WinViewport2D()
	vp.SetProp("background-color", "#FFF")
	vp.Fill = true

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	trow := vlay.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutRow
	trow.SetProp("align-vert", gi.AlignMiddle)
	trow.SetProp("align-horiz", "center")
	trow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	trow.SetStretchMaxWidth()

	spc := vlay.AddNewChild(gi.KiT_Space, "spc1").(*gi.Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	trow.AddNewChild(gi.KiT_Stretch, "str1")
	lab1 := trow.AddNewChild(gi.KiT_Label, "lab1").(*gi.Label)
	lab1.Text = "This is a test of the Slice and Map Views reflect-ive GUI"
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	trow.AddNewChild(gi.KiT_Stretch, "str2")

	split := vlay.AddNewChild(gi.KiT_SplitView, "split").(*gi.SplitView)
	split.Dim = gi.X

	mvfr := split.AddNewChild(gi.KiT_Frame, "mvfr").(*gi.Frame)
	svfr := split.AddNewChild(gi.KiT_Frame, "svfr").(*gi.Frame)
	split.SetSplits(.5, .5)

	mv := mvfr.AddNewChild(gi.KiT_MapView, "mv").(*gi.MapView)
	mv.SetMap(&tstmap)

	sv := svfr.AddNewChild(gi.KiT_SliceView, "sv").(*gi.SliceView)
	sv.SetSlice(&tstslice)

	win.UpdateEnd()

	win.StartEventLoop()
}
