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
		s.SetAbilities(true, abilities.Activatable, abilities.Slideable)
		s.SetStretchMax()
	})
}

func (se *Scene3D) HandleScene3DEvents() {
	se.On(events.MouseDown, func(e events.Event) {
		se.Scene.MouseDownEvent(e)
	})
	se.On(events.SlideMove, func(e events.Event) {
		se.Scene.SlideMoveEvent(e)
	})
	se.On(events.Scroll, func(e events.Event) {
		se.Scene.MouseScrollEvent(e.(*events.MouseScroll))
	})
	se.On(events.KeyChord, func(e events.Event) {
		se.Scene.KeyChordEvent(e)
	})
	se.HandleWidgetEvents()
}

func (se *Scene3D) GetSize(sc *gi.Scene, iter int) {
	se.InitLayout(sc)
	zp := image.Point{}
	if se.Scene.Geom.Size != zp {
		se.GetSizeFromWH(float32(se.Scene.Geom.Size.X), float32(se.Scene.Geom.Size.Y))
	} else {
		se.GetSizeFromWH(640, 480)
	}
}

func (se *Scene3D) ConfigWidget(sc *gi.Scene) {
	se.Sc = sc
	se.ConfigFrame(sc)
}

// ConfigFrame configures the framebuffer for GPU rendering,
// using the RenderWin GPU and Device.
func (se *Scene3D) ConfigFrame(sc *gi.Scene) {
	zp := image.Point{}
	sz := se.LayState.Alloc.Size.ToPoint()
	if sz == zp {
		return
	}
	se.Scene.Geom.Size = sz
	fmt.Println("sz", sz)

	doConfig := false
	if se.Scene.Frame != nil {
		cursz := se.Scene.Frame.Format.Size
		if cursz == sz {
			fmt.Println("cursz == sz")
			return
		}
	} else {
		doConfig = true
	}

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
	pos := se.LayState.Alloc.Pos.ToPointCeil()
	max := pos.Add(se.LayState.Alloc.Size.ToPointCeil())
	r := image.Rectangle{Min: pos, Max: max}
	sp := image.Point{}
	if se.Par != nil { // use parents children bbox to determine where we can draw
		pni, _ := gi.AsWidget(se.Par)
		pbb := pni.ChildrenBBoxes(sc)
		nr := r.Intersect(pbb)
		sp = nr.Min.Sub(r.Min)
		if sp.X < 0 || sp.Y < 0 || sp.X > 10000 || sp.Y > 10000 {
			fmt.Printf("Scene3D aberrant sp: %v\n", sp)
			return
		}
		r = nr
	}
	img, err := se.Scene.Image()
	if err != nil {
		log.Println("frame image err:", err)
		return
	}
	// fmt.Println("r", r, "bnd", img.Bounds())
	draw.Draw(sc.Pixels, r, img, sp, draw.Over)
}

// Render3D renders the Frame Image
func (se *Scene3D) Render3D(sc *gi.Scene) {
	se.ConfigFrame(sc)
	if se.Scene.Frame == nil {
		return
	}
	se.Scene.UpdateNodes()
	se.Scene.Render()
}

func (se *Scene3D) Render(sc *gi.Scene) {
	if se.PushBounds(sc) {
		se.RenderChildren(sc)
		se.Render3D(sc)
		se.DrawIntoScene(sc)
		se.PopBounds(sc)
	}
}

// Direct render to Drawer frame
// drw := sc.Win.OSWin.Drawer()
// drw.SetFrameImage(sc.DirUpIdx, sc.Frame.Frames[0])
// sc.Win.DirDraws.SetWinBBox(sc.DirUpIdx, sc.WinBBox)
// drw.SyncImages()
