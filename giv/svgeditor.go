// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/abilities"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/mat32/v2"
	"goki.dev/svg"
)

// SVGEditor supports editing of SVG elements
type SVGEditor struct {
	gi.SVG

	// view translation offset (from dragging)
	Trans mat32.Vec2

	// view scaling (from zooming)
	Scale float32

	// has dragging cursor been set yet?
	SetDragCursor bool `view:"-"`
}

func (sve *SVGEditor) CopyFieldsFrom(frm any) {
	fr := frm.(*SVGEditor)
	sve.SVG.CopyFieldsFrom(&fr.SVG)
	sve.Trans = fr.Trans
	sve.Scale = fr.Scale
	sve.SetDragCursor = fr.SetDragCursor
}

func (sve *SVGEditor) OnInit() {
	sve.SVG.OnInit()
	// todo: abilities include draggable
	sve.HandleEvents()
	sve.SetStyles()
}

func (sve *SVGEditor) SetStyles() {
	sve.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Slideable, abilities.Pressable, abilities.LongHoverable, abilities.Scrollable)
	})
}

func (sve *SVGEditor) ContextMenu(m *gi.Scene) {
	gi.NewButton(m).SetText("Edit").SetIcon(icons.Edit).
		SetTooltip("edit object(s) under mouse").
		OnClick(func(e events.Event) {
			objs := svg.NodesContainingPoint(&sve.SVG.SVG.Root, e.LocalPos(), true)
			if len(objs) == 0 {
				gi.MessageSnackbar(sve, "no svg nodes found under mouse")
				return
			}
			if len(objs) == 1 {
				d := gi.NewBody().AddTitle(sve.Nm + " Node")
				NewStructView(d).SetStruct(objs[0])
				d.NewFullDialog(sve).Run()
			} else {
				d := gi.NewBody().AddTitle(sve.Nm + " Nodes")
				NewSliceView(d).SetSlice(objs)
				d.NewFullDialog(sve).Run()
			}
		})
}

func (sve *SVGEditor) HandleEvents() {
	sve.On(events.SlideMove, func(e events.Event) {
		e.SetHandled()
		del := e.PrevDelta()
		sve.Trans.X += float32(del.X)
		sve.Trans.Y += float32(del.Y)
		sve.SetTransform()
		sve.SetNeedsRender(true)
	})
	sve.On(events.Scroll, func(e events.Event) {
		e.SetHandled()
		se := e.(*events.MouseScroll)
		sve.InitScale()
		sve.Scale += float32(se.DimDelta(mat32.Y)) / 20
		if sve.Scale <= 0 {
			sve.Scale = 0.01
		}
		sve.SetTransform()
		sve.SetNeedsRender(true)
	})
	// sve.AddFunc(events.LongHoverStart, RegPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	e := d.(events.Event)
	// 	e.SetHandled()
	// 	sve := sve
	// 	obj := sve.FirstContainingPoint(e.LocalPos(), true)
	// 	if obj != nil {
	// 		pos := e.LocalPos()
	// 		ttxt := fmt.Sprintf("element name: %v -- use right mouse click to edit", obj.Name())
	// 		PopupTooltip(obj.Name(), pos.X, pos.Y, sve.Sc, ttxt)
	// 	}
	// })
}

// InitScale ensures that Scale is initialized and non-zero
func (sve *SVGEditor) InitScale() {
	if sve.Scale == 0 {
		mvp := sve.Sc
		if mvp != nil {
			sve.Scale = 1 // todo: sve.ParentRenderWin().LogicalDPI() / 96.0
		} else {
			sve.Scale = 1
		}
	}
}

// SetTransform sets the transform based on Trans and Scale values
func (sve *SVGEditor) SetTransform() {
	sve.InitScale()
	if sve.SVG.SVG != nil {
		sve.SVG.SVG.Norm = false
		sve.SVG.SVG.Root.SetProp("transform", fmt.Sprintf("translate(%v,%v) scale(%v,%v)", sve.Trans.X, sve.Trans.Y, sve.Scale, sve.Scale))
	}
}
