// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3dv

//go:generate goki generate

import (
	"fmt"
	"image"
	"image/draw"
	"log"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi3d"
	"goki.dev/girl/abilities"
	"goki.dev/girl/styles"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
)

// Scene3D contains a svg.SVG element.
// The rendered version is cached for a given size.
type Scene3D struct {
	gi.WidgetBase

	// Scene is the 3D Scene
	Scene gi3d.Scene
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
		s.SetStretchMax()
	})
}

func (se *Scene3D) HandleScene3DEvents() {
	se.On(events.MouseDown, func(e events.Event) {
		se.Scene.MouseDownEvent(e)
		se.SetNeedsRender()
	})
	se.On(events.SlideMove, func(e events.Event) {
		se.Scene.SlideMoveEvent(e)
		se.SetNeedsRender()
	})
	se.On(events.Scroll, func(e events.Event) {
		se.Scene.MouseScrollEvent(e.(*events.MouseScroll))
		se.SetNeedsRender()
	})
	se.On(events.KeyChord, func(e events.Event) {
		se.Scene.KeyChordEvent(e)
		se.SetNeedsRender()
	})
	se.HandleWidgetEvents()
}

func (se *Scene3D) GetSize(sc *gi.Scene, iter int) {
	se.InitLayout(sc)
	se.GetSizeFromWH(16, 16) // minimal size
}

func (se *Scene3D) ConfigWidget(sc *gi.Scene) {
	se.Sc = sc
	se.ConfigFrame(sc)
}

// ConfigFrame configures the framebuffer for GPU rendering,
// using the RenderWin GPU and Device.
func (se *Scene3D) ConfigFrame(sc *gi.Scene) {
	zp := image.Point{}
	sz := se.LayState.Alloc.Size.ToPointFloor()
	if sz == zp {
		return
	}
	// requires 4-wise alignment apparently?
	sz.X -= sz.X % 4
	// sz.Y -= sz.Y%4
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
	fmt.Println("new frame sz", sz)

	win := sc.EventMgr.RenderWin()
	if win == nil {
		return
	}
	drw := win.GoosiWin.Drawer()
	goosi.TheApp.RunOnMain(func() {
		se.Scene.ConfigFrameFromDrawer(drw)
		if doConfig {
			se.Scene.Config()
		}
	})
	se.Scene.SetFlag(true, gi3d.ScNeedsRender)
	se.SetNeedsRender()
}

func (se *Scene3D) ApplyStyle(sc *gi.Scene) {
	se.StyMu.Lock()
	defer se.StyMu.Unlock()

	se.ApplyStyleWidget(sc)
}

func (se *Scene3D) DoLayout(sc *gi.Scene, parBBox image.Rectangle, iter int) bool {
	se.DoLayoutBase(sc, parBBox, iter)
	return se.DoLayoutChildren(sc, iter)
}

func (se *Scene3D) DrawIntoScene(sc *gi.Scene) {
	if se.Scene.Frame == nil {
		return
	}
	pos := se.LayState.Alloc.Pos.ToPointFloor()
	sz := se.LayState.Alloc.Size.ToPointCeil()
	// sz := se.Scene.Geom.Size
	max := pos.Add(sz)
	r := image.Rectangle{Min: pos, Max: max}
	sp := image.Point{}
	if se.Par != nil { // use parents children bbox to determine where we can draw
		pni, _ := gi.AsWidget(se.Par)
		pbb := pni.ChildrenBBoxes(sc)
		nr := r.Intersect(pbb)
		sp = nr.Min.Sub(r.Min)
		if sp.X < 0 || sp.Y < 0 || sp.X > 10000 || sp.Y > 10000 {
			fmt.Println("Scene3D aberrant sp:", sp, "r:", r, "pbb:", pbb)
			return
		}
		r = nr
	}
	img, err := se.Scene.Image() // direct access
	if err != nil {
		log.Println("frame image err:", err)
		return
	}
	// fmt.Println("r", r, "bnd", img.Bounds())
	draw.Draw(sc.Pixels, r, img, sp, draw.Src) // note: critical to not use Over here!
	se.Scene.ImageDone()
}

// Render3D renders the Frame Image
func (se *Scene3D) Render3D(sc *gi.Scene) {
	se.ConfigFrame(sc) // nop if all good
	if se.Scene.Frame == nil {
		return
	}
	if se.Scene.Is(gi3d.ScNeedsConfig) {
		goosi.TheApp.RunOnMain(func() {
			se.Scene.Config()
		})
	}
	se.Scene.DoUpdate()
}

func (se *Scene3D) Render(sc *gi.Scene) {
	if se.PushBounds(sc) {
		se.Render3D(sc)
		se.DrawIntoScene(sc)
		se.RenderChildren(sc)
		se.PopBounds(sc)
	}
}

// Direct render to Drawer frame
// drw := sc.Win.OSWin.Drawer()
// drw.SetFrameImage(sc.DirUpIdx, sc.Frame.Frames[0])
// sc.Win.DirDraws.SetWinBBox(sc.DirUpIdx, sc.WinBBox)
// drw.SyncImages()
