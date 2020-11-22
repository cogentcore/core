// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Originally written by Kai O'Reilly (https://github.com/kplat1) with some help from his dad..

package main

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/svg"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
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
	gimain.Main(func() {
		mainrun()
	})
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
			"background-color": gist.Color{240, 255, 240, 255},
		},
		".hslides": ki.Props{
			"background-color": gist.Color{240, 225, 255, 255},
		},
		"kbd": ki.Props{
			"color": "blue",
		},
	}
	// Vp.CSS = css
	_ = css

	mfr := win.SetMainFrame()

	// the StructView will also show the Graph Toolbar which is main actions..
	gstru := giv.AddNewStructView(mfr, "gstru")
	gstru.Viewport = Vp // needs vp early for toolbar
	gstru.SetProp("height", "4.5em")
	gstru.SetStruct(&Gr)
	ParamsEdit = gstru

	lns := giv.AddNewTableView(mfr, "lns")
	lns.Viewport = Vp
	lns.SetSlice(&Gr.Lines)
	EqTable = lns

	frame := gi.AddNewFrame(mfr, "frame", gi.LayoutHoriz)

	SvgGraph = svg.AddNewSVG(frame, "graph")
	SvgGraph.SetProp("min-width", GraphSize)
	SvgGraph.SetProp("min-height", GraphSize)
	SvgGraph.SetStretchMaxWidth()
	SvgGraph.SetStretchMaxHeight()

	SvgLines = svg.AddNewGroup(SvgGraph, "SvgLines")
	SvgMarbles = svg.AddNewGroup(SvgGraph, "SvgMarbles")
	SvgCoords = svg.AddNewGroup(SvgGraph, "SvgCoords")

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
