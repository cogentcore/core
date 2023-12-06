// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzv

//go:generate goki generate

import (
	"image"
	"image/draw"
	"log"
	"log/slog"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/abilities"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/vgpu/v2/vgpu"
	"goki.dev/xyz"
)

// Scene3D contains a svg.SVG element.
// The rendered version is cached for a given size.
type Scene3D struct {
	gi.WidgetBase

	// Scene is the 3D Scene
	Scene xyz.Scene
}

func (se *Scene3D) CopyFieldsFrom(frm any) {
	fr := frm.(*Scene3D)
	se.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	se.Scene.CopyFrom(&fr.Scene)
}

func (se *Scene3D) OnInit() {
	se.Scene.InitName(&se.Scene, "Scene")
	se.Scene.Defaults()
	se.HandleScene3DEvents()
	se.Scene3DStyles()
}

func (se *Scene3D) Scene3DStyles() {
	se.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Focusable, abilities.Activatable, abilities.Slideable)
		s.Grow.Set(1, 1)
		s.Min.Set(units.Em(20))
	})
}

func (se *Scene3D) HandleScene3DEvents() {
	se.On(events.MouseDown, func(e events.Event) {
		se.Scene.MouseDownEvent(e)
		se.SetNeedsRender(true)
	})
	se.On(events.SlideMove, func(e events.Event) {
		se.Scene.SlideMoveEvent(e)
		se.SetNeedsRender(true)
	})
	se.On(events.Scroll, func(e events.Event) {
		se.Scene.MouseScrollEvent(e.(*events.MouseScroll))
		se.SetNeedsRender(true)
	})
	se.On(events.KeyChord, func(e events.Event) {
		se.Scene.KeyChordEvent(e)
		se.SetNeedsRender(true)
	})
	se.HandleWidgetEvents()
}

func (se *Scene3D) ConfigWidget() {
	se.ConfigFrame()
}

// ConfigFrame configures the framebuffer for GPU rendering,
// using the RenderWin GPU and Device.
func (se *Scene3D) ConfigFrame() {
	zp := image.Point{}
	sz := se.Geom.Size.Actual.Content.ToPointFloor()
	if sz == zp {
		return
	}
	se.Scene.Geom.Size = sz

	doConfig := false
	if se.Scene.Frame != nil {
		cursz := se.Scene.Frame.Format.Size
		if cursz == sz {
			return
		}
	} else {
		doConfig = true
	}

	win := se.Sc.EventMgr.RenderWin()
	if win == nil {
		return
	}
	drw := win.GoosiWin.Drawer()
	goosi.TheApp.RunOnMain(func() {
		se.Scene.ConfigFrameFromSurface(drw.Surface().(*vgpu.Surface))
		if doConfig {
			se.Scene.Config()
		}
	})
	se.Scene.SetFlag(true, xyz.ScNeedsRender)
	se.SetNeedsRender(true)
}

func (se *Scene3D) DrawIntoScene() {
	if se.Scene.Frame == nil {
		return
	}
	r := se.Geom.ContentBBox
	sp := image.Point{}
	if se.Par != nil { // use parents children bbox to determine where we can draw
		_, pwb := gi.AsWidget(se.Par)
		pbb := pwb.Geom.ContentBBox
		nr := r.Intersect(pbb)
		sp = nr.Min.Sub(r.Min)
		if sp.X < 0 || sp.Y < 0 || sp.X > 10000 || sp.Y > 10000 {
			slog.Error("xyzv.Scene3D bad bounding box", "path", se, "startPos", sp, "bbox", r, "parBBox", pbb)
			return
		}
		r = nr
	}
	img, err := se.Scene.Image() // direct access
	if err != nil {
		log.Println("frame image err:", err)
		return
	}
	draw.Draw(se.Sc.Pixels, r, img, sp, draw.Src) // note: critical to not use Over here!
	se.Scene.ImageDone()
}

// Render renders the Frame Image
func (se *Scene3D) Render3D() {
	se.ConfigFrame() // nop if all good
	if se.Scene.Frame == nil {
		return
	}
	if se.Scene.Is(xyz.ScNeedsConfig) {
		goosi.TheApp.RunOnMain(func() {
			se.Scene.Config()
		})
	}
	se.Scene.DoUpdate()
}

func (se *Scene3D) Render() {
	if se.PushBounds() {
		se.Render3D()
		se.DrawIntoScene()
		se.RenderChildren()
		se.PopBounds()
	}
}

// UpdateStart3D calls UpdateStart on the 3D Scene:
// sets the scene ScUpdating flag to prevent
// render updates during construction on a scene.
// if already updating, returns false.
// Pass the result to UpdateEnd* methods.
func (se *Scene3D) UpdateStart3D() bool {
	return se.Scene.UpdateStart()
}

// UpdateEnd3D calls UpdateEnd on the 3D Scene:
// resets the scene ScUpdating flag if updt = true
func (se *Scene3D) UpdateEnd3D(updt bool) {
	se.Scene.UpdateEnd(updt)
}

// UpdateEndRender3D calls UpdateEndRender on the 3D Scene
// and calls gi SetNeedsRender.
// resets the scene ScUpdating flag if updt = true
// and sets the ScNeedsRender flag; updt is from UpdateStart().
// Render only updates based on camera changes, not any node-level
// changes. See [UpdateEndUpdate].
func (se *Scene3D) UpdateEndRender3D(updt bool) {
	if updt {
		se.Scene.UpdateEndRender(updt)
		se.SetNeedsRender(updt)
	}
}

// UpdateEndUpdate3D calls UpdateEndUpdate on the 3D Scene
// and calls gi SetNeedsRender.
// UpdateEndUpdate resets the scene ScUpdating flag if updt = true
// and sets the ScNeedsUpdate flag; updt is from UpdateStart().
// Update is for when any node Pose or material changes happen.
// See [UpdateEndConfig] for major changes.
func (se *Scene3D) UpdateEndUpdate3D(updt bool) {
	if updt {
		se.Scene.UpdateEndUpdate(updt)
		se.SetNeedsRender(updt)
	}
}

// UpdateEndConfig3D calls UpdateEndConfig on the 3D Scene
// and calls gi SetNeedsRender.
// UpdateEndConfig resets the scene ScUpdating flag if updt = true
// and sets the ScNeedsConfig flag; updt is from UpdateStart().
// Config is for Texture, Lighting Meshes or more complex nodes).
func (se *Scene3D) UpdateEndConfig3D(updt bool) {
	if updt {
		se.Scene.UpdateEndConfig(updt)
		se.SetNeedsRender(updt)
	}
}

// Direct render to Drawer frame
// drw := sc.Win.OSWin.Drawer()
// drw.SetFrameImage(sc.DirUpIdx, sc.Frame.Frames[0])
// sc.Win.DirDraws.SetWinBBox(sc.DirUpIdx, sc.WinBBox)
// drw.SyncImages()
