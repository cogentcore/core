// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/rcoreilly/goki/gi"
	"github.com/skelterjohn/go.wde"
	_ "github.com/skelterjohn/go.wde/init"
	"math/rand"
	"reflect"
	"runtime"
	"sync"
	// "time"
	// "image"
	// "image/draw"
)

func main() {
	go mainrun()
	wde.Run() // this needs to run in main loop
}

func mainrun() {
	var wg sync.WaitGroup

	x := func() {
		width := 400
		height := 400
		win := gi.NewWindow2D("test window", width, height)

		vp := win.WinViewport2D()

		// rect1.SetProp("stroke-linejoin", "round")
		rect1 := vp.AddNewChildNamed(reflect.TypeOf(gi.GiRect{}), "rect1").(*gi.GiRect)
		rect1.SetProp("fill", "#008800")
		rect1.SetProp("stroke", "#0000FF")
		rect1.SetProp("stroke-width", 5.0)
		rect1.Pos = gi.Point2D{10, 10}
		rect1.Size = gi.Size2D{100, 100}

		circle1 := vp.AddNewChildNamed(reflect.TypeOf(gi.GiCircle{}), "circle1").(*gi.GiCircle)
		circle1.SetProp("fill", "none") // todo: need to process
		circle1.SetProp("stroke", "#CC0000")
		circle1.SetProp("stroke-width", 2.0)
		circle1.Pos = gi.Point2D{200, 100}
		circle1.Radius = 40

		ellipse1 := circle1.AddNewChildNamed(reflect.TypeOf(gi.GiEllipse{}), "ellipse1").(*gi.GiEllipse)
		ellipse1.SetProp("fill", "#55000055")
		ellipse1.SetProp("stroke", "#880000")
		ellipse1.SetProp("stroke-width", 2.0)
		ellipse1.Pos = gi.Point2D{100, 100}
		ellipse1.Radii = gi.Size2D{80, 20}

		line1 := vp.AddNewChildNamed(reflect.TypeOf(gi.GiLine{}), "line1").(*gi.GiLine)
		line1.SetProp("stroke", "#888800")
		line1.SetProp("stroke-width", 5.0)
		line1.Start = gi.Point2D{100, 100}
		line1.End = gi.Point2D{150, 200}

		polyline1 := vp.AddNewChildNamed(reflect.TypeOf(gi.GiPolyline{}), "polyline1").(*gi.GiPolyline)
		polyline1.SetProp("stroke", "#888800")
		polyline1.SetProp("stroke-width", 4.0)

		for i := 0; i < 10; i++ {
			x1 := rand.Float64() * float64(width)
			y1 := rand.Float64() * float64(height)
			polyline1.Points = append(polyline1.Points, gi.Point2D{x1, y1})
		}

		polygon1 := vp.AddNewChildNamed(reflect.TypeOf(gi.GiPolygon{}), "polygon1").(*gi.GiPolygon)
		polygon1.SetProp("fill", "#55005555")
		polygon1.SetProp("stroke", "#888800")
		polygon1.SetProp("stroke-width", 4.0)

		for i := 0; i < 10; i++ {
			x1 := rand.Float64() * float64(width)
			y1 := rand.Float64() * float64(height)
			polygon1.Points = append(polygon1.Points, gi.Point2D{x1, y1})
		}

		vp.InitTopLevel()
		vp.Clear()
		// vp.RenderTopLevel()
		polygon1.UpdateStart()
		polygon1.UpdateEnd(false) // only update highest

		events := win.Win.EventChan()

		done := make(chan bool)

		go func() {
		loop:
			for ei := range events {
				runtime.Gosched()
				switch e := ei.(type) {
				case wde.MouseDownEvent:
					fmt.Println("clicked", e.Where.X, e.Where.Y, e.Which)
					polygon1.UpdateStart()
					polygon1.UpdateEnd(false) // only update highest
					// win.Win.Close()
					// break loop
				case wde.MouseUpEvent:
				case wde.MouseMovedEvent:
				case wde.MouseDraggedEvent:
				case wde.MouseEnteredEvent:
					fmt.Println("mouse entered", e.Where.X, e.Where.Y)
				case wde.MouseExitedEvent:
					fmt.Println("mouse exited", e.Where.X, e.Where.Y)
				case wde.MagnifyEvent:
					fmt.Println("magnify", e.Where, e.Magnification)
				case wde.RotateEvent:
					fmt.Println("rotate", e.Where, e.Rotation)
				case wde.ScrollEvent:
					fmt.Println("scroll", e.Where, e.Delta)
				case wde.KeyDownEvent:
					// fmt.Println("KeyDownEvent", e.Glyph)
				case wde.KeyUpEvent:
					// fmt.Println("KeyUpEvent", e.Glyph)
				case wde.KeyTypedEvent:
					fmt.Printf("typed key %v, glyph %v, chord %v\n", e.Key, e.Glyph, e.Chord)
					switch e.Glyph {
					case "1":
						win.Win.SetCursor(wde.NormalCursor)
					case "2":
						win.Win.SetCursor(wde.CrosshairCursor)
					case "3":
						win.Win.SetCursor(wde.GrabHoverCursor)
					}
				case wde.CloseEvent:
					fmt.Println("close")
					win.Win.Close()
					break loop
				case wde.ResizeEvent:
					fmt.Println("resize", e.Width, e.Height)
				}
			}
			done <- true
			fmt.Println("end of events")
			polygon1.UpdateStart()
			polygon1.UpdateEnd(false) // only update highest
		}()
	}

	wg.Add(1)
	go x()
	// wg.Add(1)
	// go x()

	wg.Wait()
	wde.Stop()
}
