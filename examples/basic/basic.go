// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	// "fmt"
	"github.com/rcoreilly/goki/gi"
	_ "github.com/rcoreilly/goki/gi/init"
	// "math/rand"
	// "reflect"
	// "runtime"
	// "sync"
	// "time"
	// "image"
	// "image/draw"
)

func main() {
	go mainrun()
	gi.RunBackendEventLoop() // this needs to run in main loop
}

func mainrun() {
	width := 800
	height := 800
	win := gi.NewWindow2D("test window", width, height)
	win.UpdateStart()

	vp := win.WinViewport2D()

	vpfill := vp.AddNewChildNamed(gi.KiT_Viewport2DFill, "vpfill").(*gi.Viewport2DFill)
	vpfill.SetProp("fill", "#FFF")

	// rect1.SetProp("stroke-linejoin", "round")
	rect1 := vpfill.AddNewChildNamed(gi.KiT_Rect, "rect1").(*gi.Rect)
	rect1.SetProp("fill", "#008800")
	rect1.SetProp("stroke", "#0000FF")
	rect1.SetProp("stroke-width", 5.0)
	rect1.Pos = gi.Vec2D{10, 10}
	rect1.Size = gi.Vec2D{100, 100}

	circle1 := vpfill.AddNewChildNamed(gi.KiT_Circle, "circle1").(*gi.Circle)
	circle1.SetProp("fill", "none")
	circle1.SetProp("stroke", "#CC0000")
	circle1.SetProp("stroke-width", 2.0)
	circle1.Pos = gi.Vec2D{400, 400}
	circle1.Radius = 40

	ellipse1 := circle1.AddNewChildNamed(gi.KiT_Ellipse, "ellipse1").(*gi.Ellipse)
	ellipse1.SetProp("fill", "#55000055")
	ellipse1.SetProp("stroke", "#880000")
	ellipse1.SetProp("stroke-width", 2.0)
	ellipse1.Pos = gi.Vec2D{400, 200}
	ellipse1.Radii = gi.Vec2D{80, 20}

	line1 := vpfill.AddNewChildNamed(gi.KiT_Line, "line1").(*gi.Line)
	line1.SetProp("stroke", "#888800")
	line1.SetProp("stroke-width", 5.0)
	line1.Start = gi.Vec2D{100, 100}
	line1.End = gi.Vec2D{150, 200}

	text1 := vpfill.AddNewChildNamed(gi.KiT_Text2D, "text1").(*gi.Text2D)
	text1.SetProp("stroke", "#000")
	text1.SetProp("stroke-width", 1.0)
	text1.SetProp("text-align", "left")
	text1.SetProp("font-size", 32)
	// text1.SetProp("font-face", "Times New Roman")
	text1.SetProp("font-face", "Arial")
	text1.Pos = gi.Vec2D{10, 600}
	text1.Width = 100
	text1.Text = "this is test text!"

	rlay := vpfill.AddNewChildNamed(gi.KiT_Layout, "rowlay").(*gi.Layout)
	rlay.Lay = gi.LayoutRow
	rlay.SetProp("x", 100)
	rlay.SetProp("y", 500)
	rlay.SetProp("text-align", "center")
	button1 := rlay.AddNewChildNamed(gi.KiT_Button, "button1").(*gi.Button)
	button2 := rlay.AddNewChildNamed(gi.KiT_Button, "button2").(*gi.Button)

	button1.Text = "Button 1"
	button2.Text = "Button 2"

	win.UpdateEnd()

	win.StartEventLoop()
}
