// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/goosi/events"
	"goki.dev/mat32/v2"
)

// todo: this should just create an SVG as first child?

// Editor supports editing of SVG elements
type Editor struct {
	gi.WidgetBase

	// view translation offset (from dragging)
	Trans mat32.Vec2

	// view scaling (from zooming)
	Scale float32

	// has dragging cursor been set yet?
	SetDragCursor bool `view:"-"`
}

func (sve *Editor) CopyFieldsFrom(frm any) {
	fr := frm.(*Editor)
	// g.SVG.CopyFieldsFrom(&fr.SVG)
	sve.Trans = fr.Trans
	sve.Scale = fr.Scale
	sve.SetDragCursor = fr.SetDragCursor
}

func (sve *Editor) OnInit() {
	// todo: abilities include draggable
	sve.HandleEditorEvents()
}

// HandleEditorEvents handles svg editing events
func (sve *Editor) HandleEditorEvents() {
	sve.On(events.DragMove, func(e events.Event) {
		e.SetHandled()
		del := e.PrevDelta()
		sve.Trans.X += float32(del.X)
		sve.Trans.Y += float32(del.Y)
		sve.SetTransform()
	})
	sve.On(events.Scroll, func(e events.Event) {
		e.SetHandled()
		sve.InitScale()
		// todo:
		// sve.Scale += float32(e.ScrollDelta(mat32.Y)) / 20
		if sve.Scale <= 0 {
			sve.Scale = 0.01
		}
		sve.SetTransform()
		// ssvg.SetFullReRender()
		// ssvg.UpdateSig()
	})
	// todo: context menu
	// sve.OnAddFunc(events.MouseUp, RegPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	obj := sve.FirstContainingPoint(e.LocalPos(), true)
	// 	_ = obj
	// 	if e.Action == events.Release && e.Button == events.Right {
	// 		e.SetHandled()
	// 		// if obj != nil {
	// 		// 	giv.StructViewDialog(ssvg.Scene, obj, giv.DlgOpts{Title: "SVG Element View"}, nil, nil)
	// 		// }
	// 	}
	// })
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
func (sve *Editor) InitScale() {
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
func (sve *Editor) SetTransform() {
	sve.InitScale()
	sve.SetProp("transform", fmt.Sprintf("translate(%v,%v) scale(%v,%v)", sve.Trans.X, sve.Trans.Y, sve.Scale, sve.Scale))
}

func (sve *Editor) Render(sc *gi.Scene) {
	if sve.PushBounds(sc) {
		// rs := &sve.Render
		// if sve.Fill {
		// 	sve.FillScene()
		// }
		// if sve.Norm {
		// 	sve.SetNormXForm()
		// }
		// rs.PushXForm(sve.Pnt.XForm)
		sve.RenderChildren(sc) // we must do children first, then us!
		sve.PopBounds(sc)
		// rs.PopXForm()
		// fmt.Printf("geom.bounds: %v  geom: %v\n", svg.Geom.Bounds(), svg.Geom)
	}
}
