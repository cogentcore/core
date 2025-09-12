// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"
	"image"
	"strings"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
)

var (
	OrbitFactor = float32(0.025)
	PanFactor   = float32(0.001)
)

// NodesUnderPoint returns list of nodes within given node that
// have their ScBBox within given 2D scene point (excludes starting node).
// This is a good first-pass step for node-level
// event handling based on 2D mouse events.
func NodesUnderPoint(n tree.Node, pt image.Point) []Node {
	var ns []Node
	n.AsTree().WalkDown(func(cn tree.Node) bool {
		if cn == n {
			return tree.Continue
		}
		ni, nb := AsNode(cn)
		if !ni.IsVisible() {
			return tree.Break
		}
		if pt.In(nb.SceneBBox) {
			sbwd := nb.SceneBBox.Size().X
			if nb.isLinear {
				sp := math32.Vec3(0, 0, 0)
				ep := math32.Vec3(1, 0, 0)
				spr := sp.MulMatrix4(&nb.Pose.MVPMatrix)
				epr := ep.MulMatrix4(&nb.Pose.MVPMatrix)
				del := epr.Sub(spr)
				angxy := math32.RadToDeg(math32.Atan2(del.Y, del.X))
				if sbwd < 5 && ((angxy >= 89 && angxy <= 91) || (angxy >= -91 && angxy <= -89)) {
					ns = append(ns, ni)
					return tree.Continue
				}
				mn := math32.FromPoint(nb.SceneBBox.Min)
				mx := math32.FromPoint(nb.SceneBBox.Max)
				st := mn
				ed := mx
				flip := (angxy > 0 && angxy < 90) || (angxy < 0 && angxy < -90)
				if flip {
					st = math32.Vec2(mn.X, mx.Y)
					ed = math32.Vec2(mx.X, mn.Y)
				}
				ln := math32.NewLine2(st, ed)
				pos := math32.FromPoint(pt)
				cpp := ln.ClosestPointToPoint(pos)
				dst := cpp.Sub(pos).Length()
				if dst < 10 { // pixels
					ns = append(ns, ni)
				}
				return tree.Continue
			}
			ns = append(ns, ni)
		}
		return tree.Continue
	})
	return ns
}

func (sc *Scene) SlideMoveEvent(e events.Event) {
	cdist := math32.Max(sc.Camera.DistanceTo(sc.Camera.Target), 1.0)
	orbDel := OrbitFactor * cdist
	panDel := PanFactor * cdist

	del := e.PrevDelta()
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
		if math32.Abs(dx) > math32.Abs(dy) {
			dy = 0
		} else {
			dx = 0
		}
		sc.Camera.Orbit(-dx*orbDel, -dy*orbDel)
	}
}

func (sc *Scene) MouseScrollEvent(e *events.MouseScroll) {
	if sc.NoNav {
		return
	}
	e.SetHandled()
	pt := e.Pos()
	sz := sc.Geom.Size
	cdist := math32.Max(sc.Camera.DistanceTo(sc.Camera.Target), 1.0)
	zoom := float32(e.Delta.Y) // float32(e.ScrollNonZeroDelta(false))
	zoomDel := float32(.02) * cdist
	switch {
	case e.HasAllModifiers(key.Alt):
		sc.Camera.PanTarget(0, 0, zoom*zoomDel)
	default:
		sc.Camera.ZoomTo(pt, sz, zoom*zoomDel)
	}
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
		return
	}
}
