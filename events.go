// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"strings"

	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/mat32/v2"
)

func (sc *Scene) SlideMoveEvent(e events.Event) {
	cdist := mat32.Max(sc.Camera.DistTo(sc.Camera.Target), 1.0)
	orbDel := 0.025 * cdist
	panDel := 0.001 * cdist

	del := e.StartDelta()
	dx := float32(del.X)
	dy := float32(del.Y)
	switch {
	case e.HasAllModifiers(key.Shift):
		sc.Camera.Pan(dx*panDel, -dy*panDel)
	case e.HasAllModifiers(key.Control):
		sc.Camera.PanAxis(dx*panDel, -dy*panDel)
	case e.HasAllModifiers(key.Alt):
		sc.Camera.PanTarget(dx*panDel, -dy*panDel, 0)
	default:
		if mat32.Abs(dx) > mat32.Abs(dy) {
			dy = 0
		} else {
			dx = 0
		}
		sc.Camera.Orbit(-dx*orbDel, -dy*orbDel)
	}
}

// if !ssc.SetDragCursor {
// 	oswin.TheApp.Cursor(ssc.ParentWindow().OSWin).Push(cursor.HandOpen)
// 	ssc.SetDragCursor = true
// }
//
// }  else {
// if ssc.SetDragCursor {
// 	oswin.TheApp.Cursor(ssc.ParentWindow().OSWin).Pop()
// 	ssc.SetDragCursor = false
// }
// }

func (sc *Scene) MouseMoveEvent(e events.Event) {
	orbDel := float32(.2)
	panDel := float32(.05)
	del := e.StartDelta()
	dx := float32(del.X)
	dy := float32(del.Y)
	switch {
	case key.HasAllModifiers(key.Shift):
		sc.Camera.Pan(dx*panDel, -dy*panDel)
	case key.HasAllModifiers(key.Control):
		sc.Camera.PanAxis(dx*panDel, -dy*panDel)
	case key.HasAllModifiers(key.Alt):
		sc.Camera.PanTarget(dx*panDel, -dy*panDel, 0)
	default:
		if mat32.Abs(dx) > mat32.Abs(dy) {
			dy = 0
		} else {
			dx = 0
		}
		sc.Camera.Orbit(-dx*orbDel, -dy*orbDel)
	}
	// sc.UpdateSig()
}

// if sc.SetDragCursor {
// 	oswin.TheApp.Cursor(sc.ParentWindow().OSWin).Pop()
// 	sc.SetDragCursor = false
// }
// if sc.SetDragCursor {
// 	oswin.TheApp.Cursor(sc.ParentWindow().OSWin).Pop()
// 	sc.SetDragCursor = false
// }

func (sc *Scene) MouseScrollEvent(e *events.MouseScroll) {
	if sc.NoNav {
		return
	}
	pt := e.Pos() // e.Where.Sub(sc.ScBBox.Min)
	sz := sc.Geom.Size
	cdist := mat32.Max(sc.Camera.DistTo(sc.Camera.Target), 1.0)
	zoom := float32(e.DimDelta(mat32.Y)) // float32(e.ScrollNonZeroDelta(false))
	zoomDel := float32(.001) * cdist
	switch {
	case e.HasAllModifiers(key.Alt):
		sc.Camera.PanTarget(0, 0, zoom*zoomDel)
	default:
		sc.Camera.ZoomTo(pt, sz, zoom*zoomDel)
	}
	// sc.UpdateSig()
}

func (sc *Scene) MouseDownEvent(e events.Event) {
	if sc.NoNav {
		return
	}
	// if !sc.IsDisabled() && !sc.HasFocus() {
	// 	sc.GrabFocus()
	// }
	// if ssc.CurManipPt == nil {
	sc.SetSel(nil) // clear any selection at this point
	// }
}

func (sc *Scene) KeyChordEvent(e events.Event) {
	if sc.NoNav {
		return
	}
	sc.NavKeyEvents(e)
}

// NavKeyEvents handles standard viewer keyboard navigation events
func (sc *Scene) NavKeyEvents(kt events.Event) {
	ch := string(kt.KeyChord())
	// fmt.Printf(ch)
	orbDeg := float32(5)
	panDel := float32(.2)
	zoomPct := float32(.05)
	switch ch {
	case "UpArrow":
		sc.Camera.Orbit(0, orbDeg)
		kt.SetHandled()
	case "Shift+UpArrow":
		sc.Camera.Pan(0, panDel)
		kt.SetHandled()
	case "Control+UpArrow":
		sc.Camera.PanAxis(0, panDel)
		kt.SetHandled()
	case "Alt+UpArrow":
		sc.Camera.PanTarget(0, panDel, 0)
		kt.SetHandled()
	case "DownArrow":
		sc.Camera.Orbit(0, -orbDeg)
		kt.SetHandled()
	case "Shift+DownArrow":
		sc.Camera.Pan(0, -panDel)
		kt.SetHandled()
	case "Control+DownArrow":
		sc.Camera.PanAxis(0, -panDel)
		kt.SetHandled()
	case "Alt+DownArrow":
		sc.Camera.PanTarget(0, -panDel, 0)
		kt.SetHandled()
	case "LeftArrow":
		sc.Camera.Orbit(orbDeg, 0)
		kt.SetHandled()
	case "Shift+LeftArrow":
		sc.Camera.Pan(-panDel, 0)
		kt.SetHandled()
	case "Control+LeftArrow":
		sc.Camera.PanAxis(-panDel, 0)
		kt.SetHandled()
	case "Alt+LeftArrow":
		sc.Camera.PanTarget(-panDel, 0, 0)
		kt.SetHandled()
	case "RightArrow":
		sc.Camera.Orbit(-orbDeg, 0)
		kt.SetHandled()
	case "Shift+RightArrow":
		sc.Camera.Pan(panDel, 0)
		kt.SetHandled()
	case "Control+RightArrow":
		sc.Camera.PanAxis(panDel, 0)
		kt.SetHandled()
	case "Alt+RightArrow":
		sc.Camera.PanTarget(panDel, 0, 0)
		kt.SetHandled()
	case "Alt++", "Alt+=":
		sc.Camera.PanTarget(0, 0, panDel)
		kt.SetHandled()
	case "Alt+-", "Alt+_":
		sc.Camera.PanTarget(0, 0, -panDel)
		kt.SetHandled()
	case "+", "=", "Shift++":
		sc.Camera.Zoom(-zoomPct)
		kt.SetHandled()
	case "-", "_", "Shift+_":
		sc.Camera.Zoom(zoomPct)
		kt.SetHandled()
	case " ", "Escape":
		err := sc.SetCamera("default")
		if err != nil {
			sc.Camera.DefaultPose()
		}
		kt.SetHandled()
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		err := sc.SetCamera(ch)
		if err != nil {
			sc.SaveCamera(ch)
			fmt.Printf("Saved camera to: %v\n", ch)
		} else {
			fmt.Printf("Restored camera from: %v\n", ch)
		}
		kt.SetHandled()
	case "Control+0", "Control+1", "Control+2", "Control+3", "Control+4", "Control+5", "Control+6", "Control+7", "Control+8", "Control+9":
		cnm := strings.TrimPrefix(ch, "Control+")
		sc.SaveCamera(cnm)
		fmt.Printf("Saved camera to: %v\n", cnm)
		kt.SetHandled()
	case "t":
		kt.SetHandled()
		obj := sc.Child(0).(*Solid)
		fmt.Printf("updated obj: %v\n", obj.Path())
		// obj.UpdateSig()
		return
	}
	// sc.UpdateSig()
}
