// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	// "fmt"
	"github.com/rcoreilly/goki/gi"
	_ "github.com/rcoreilly/goki/gi/init"
	"math/rand"
	"reflect"
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
	width := 400
	height := 400
	win := gi.NewWindow2D("test window", width, height)
	win.UpdateStart()

	vp := win.WinViewport2D()

	bg := vp.AddNewChildNamed(reflect.TypeOf(gi.Rect{}), "bg").(*gi.Rect)
	bg.SetProp("fill", "#FFF")
	bg.Pos = gi.Point2D{0, 00}
	bg.Size = gi.Size2D{float64(width), float64(height)}

	// rect1.SetProp("stroke-linejoin", "round")
	rect1 := bg.AddNewChildNamed(reflect.TypeOf(gi.Rect{}), "rect1").(*gi.Rect)
	rect1.SetProp("fill", "#008800")
	rect1.SetProp("stroke", "#0000FF")
	rect1.SetProp("stroke-width", 5.0)
	rect1.Pos = gi.Point2D{10, 10}
	rect1.Size = gi.Size2D{100, 100}

	circle1 := bg.AddNewChildNamed(reflect.TypeOf(gi.Circle{}), "circle1").(*gi.Circle)
	circle1.SetProp("fill", "none") // todo: need to process
	circle1.SetProp("stroke", "#CC0000")
	circle1.SetProp("stroke-width", 2.0)
	circle1.Pos = gi.Point2D{200, 100}
	circle1.Radius = 40

	ellipse1 := circle1.AddNewChildNamed(reflect.TypeOf(gi.Ellipse{}), "ellipse1").(*gi.Ellipse)
	ellipse1.SetProp("fill", "#55000055")
	ellipse1.SetProp("stroke", "#880000")
	ellipse1.SetProp("stroke-width", 2.0)
	ellipse1.Pos = gi.Point2D{100, 100}
	ellipse1.Radii = gi.Size2D{80, 20}

	line1 := bg.AddNewChildNamed(reflect.TypeOf(gi.Line{}), "line1").(*gi.Line)
	line1.SetProp("stroke", "#888800")
	line1.SetProp("stroke-width", 5.0)
	line1.Start = gi.Point2D{100, 100}
	line1.End = gi.Point2D{150, 200}

	polyline1 := bg.AddNewChildNamed(reflect.TypeOf(gi.Polyline{}), "polyline1").(*gi.Polyline)
	polyline1.SetProp("stroke", "#888800")
	polyline1.SetProp("stroke-width", 4.0)

	for i := 0; i < 10; i++ {
		x1 := rand.Float64() * float64(width)
		y1 := rand.Float64() * float64(height)
		polyline1.Points = append(polyline1.Points, gi.Point2D{x1, y1})
	}

	polygon1 := bg.AddNewChildNamed(reflect.TypeOf(gi.Polygon{}), "polygon1").(*gi.Polygon)
	polygon1.SetProp("fill", "#55005555")
	polygon1.SetProp("stroke", "#888800")
	polygon1.SetProp("stroke-width", 4.0)

	for i := 0; i < 10; i++ {
		x1 := rand.Float64() * float64(width)
		y1 := rand.Float64() * float64(height)
		polygon1.Points = append(polygon1.Points, gi.Point2D{x1, y1})
	}

	bg.AddNewChildNamed(reflect.TypeOf(gi.PushButton{}), "rect1")

	vp.InitTopLevel()
	vp.Clear()
	win.UpdateEnd(false)

	polygon1.UpdateStart()
	polygon1.UpdateEnd(false)

	win.StartEventLoop()
}
