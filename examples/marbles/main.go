// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Originally written by Kai O'Reilly (https://github.com/kkoreilly)

package main

import (
	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/ki/v2/ki"
	"goki.dev/mat32/v2"
	"goki.dev/svg"
)

var Vp *gi.Viewport2D
var EqTable *giv.TableView
var ParamsEdit *giv.StructView
var SvgGraph *svg.SVG
var SvgLines *svg.Group
var SvgMarbles *svg.Group
var SvgCoords *svg.Group

var gmin, gmax, gsz, ginc mat32.Vec2
var GraphSize float32 = 800

func main() {
	gimain.Main(mainrun)
}

func mainrun() {
	width := 1024
	height := 1024

	Gr.Defaults()

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	gi.SetAppName("marbles")
	gi.SetAppAbout("marbles allows you to enter equations, which are graphed, and then marbles are dropped down on the resulting lines, and bounce around in very entertaining ways!")

	win := gi.NewMainWindow("marbles", "Marbles", width, height)

	Vp = win.WinViewport2D()
	updt := Vp.UpdateStart()

	// style sheet
	var css = ki.Props{
		"Action": ki.Props{
			"background-color": gi.Prefs.Colors.Control, // gist.Color{255, 240, 240, 255},
		},
		"#combo": ki.Props{
			"background-color": colors.FromRGB(240, 255, 240),
		},
		".hslides": ki.Props{
			"background-color": colors.FromRGB(240, 225, 255),
		},
		"kbd": ki.Props{
			"color": "blue",
		},
	}
	// Vp.CSS = css
	_ = css

	mfr := win.SetMainFrame()

	// the StructView will also show the Graph Toolbar which is main actions..
	gstru := giv.NewStructView(mfr, "gstru")
	gstru.Viewport = Vp // needs vp early for toolbar
	gstru.SetProp("height", "4.5em")
	gstru.SetStruct(&Gr)
	ParamsEdit = gstru

	lns := giv.NewTableView(mfr, "lns")
	lns.Viewport = Vp
	lns.SetSlice(&Gr.Lines)
	EqTable = lns

	frame := gi.NewFrame(mfr, "frame", gi.LayoutHoriz)

	SvgGraph = svg.NewSVG(frame, "graph")
	SvgGraph.SetProp("min-width", GraphSize)
	SvgGraph.SetProp("min-height", GraphSize)
	SvgGraph.SetStretchMaxWidth()
	SvgGraph.SetStretchMaxHeight()

	SvgLines = svg.NewGroup(SvgGraph, "SvgLines")
	SvgMarbles = svg.NewGroup(SvgGraph, "SvgMarbles")
	SvgCoords = svg.NewGroup(SvgGraph, "SvgCoords")

	gmin = mat32.Vec2{-10, -10}
	gmax = mat32.Vec2{10, 10}
	gsz = gmax.Sub(gmin)
	ginc = gsz.DivScalar(GraphSize)

	SvgGraph.ViewBox.Min = gmin
	SvgGraph.ViewBox.Size = gsz
	SvgGraph.Norm = true
	SvgGraph.InvertY = true
	SvgGraph.Fill = true
	SvgGraph.SetProp("background-color", "white")
	SvgGraph.SetProp("stroke-width", ".2pct")

	InitCoords()
	ResetMarbles()
	Gr.Lines.Graph()

	//////////////////////////////////////////
	//      Main Menu

	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	win.MainMenuUpdated()
	Vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}
