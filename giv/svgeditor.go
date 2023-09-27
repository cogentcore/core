// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"goki.dev/goosi"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/mouse"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// todo: this should just create an SVG as first child?

// Editor supports editing of SVG elements
type Editor struct {
	WidgetBase

	// view translation offset (from dragging)
	Trans mat32.Vec2 `desc:"view translation offset (from dragging)"`

	// view scaling (from zooming)
	Scale float32 `desc:"view scaling (from zooming)"`

	// [view: -] has dragging cursor been set yet?
	SetDragCursor bool `view:"-" desc:"has dragging cursor been set yet?"`
}

func (sve *Editor) CopyFieldsFrom(frm any) {
	fr := frm.(*Editor)
	// g.SVG.CopyFieldsFrom(&fr.SVG)
	sve.Trans = fr.Trans
	sve.Scale = fr.Scale
	sve.SetDragCursor = fr.SetDragCursor
}

// EditorEvents handles svg editing events
func (sve *Editor) EditorEvents() {
	svewe.AddFunc(goosi.MouseDragEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		me.SetHandled()
		ssvg := sve
		if ssvg.HasFlag(NodeDragging) {
			if !ssvg.SetDragCursor {
				goosi.TheApp.Cursor(ssvg.ParentRenderWin().RenderWin).Push(cursor.HandOpen)
				ssvg.SetDragCursor = true
			}
			del := me.Where.Sub(me.From)
			ssvg.Trans.X += float32(del.X)
			ssvg.Trans.Y += float32(del.Y)
			ssvg.SetTransform()
			// ssvg.SetFullReRender()
			// ssvg.UpdateSig()
		} else {
			if ssvg.SetDragCursor {
				goosi.TheApp.Cursor(ssvg.ParentRenderWin().RenderWin).Pop()
				ssvg.SetDragCursor = false
			}
		}

	})
	svewe.AddFunc(goosi.MouseScrollEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.ScrollEvent)
		me.SetHandled()
		ssvg := sve
		if ssvg.SetDragCursor {
			goosi.TheApp.Cursor(ssvg.ParentRenderWin().RenderWin).Pop()
			ssvg.SetDragCursor = false
		}
		ssvg.InitScale()
		ssvg.Scale += float32(me.NonZeroDelta(false)) / 20
		if ssvg.Scale <= 0 {
			ssvg.Scale = 0.01
		}
		ssvg.SetTransform()
		// ssvg.SetFullReRender()
		// ssvg.UpdateSig()
	})
	svewe.AddFunc(goosi.MouseButtonEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		ssvg := sve
		if ssvg.SetDragCursor {
			goosi.TheApp.Cursor(ssvg.ParentRenderWin().RenderWin).Pop()
			ssvg.SetDragCursor = false
		}
		obj := ssvg.FirstContainingPoint(me.Where, true)
		_ = obj
		if me.Action == mouse.Release && me.Button == mouse.Right {
			me.SetHandled()
			// if obj != nil {
			// 	giv.StructViewDialog(ssvg.Scene, obj, giv.DlgOpts{Title: "SVG Element View"}, nil, nil)
			// }
		}
	})
	svewe.AddFunc(goosi.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		me.SetHandled()
		ssvg := sve
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if obj != nil {
			pos := me.Where
			ttxt := fmt.Sprintf("element name: %v -- use right mouse click to edit", obj.Name())
			PopupTooltip(obj.Name(), pos.X, pos.Y, sve.Sc, ttxt)
		}
	})
}

func (sve *Editor) AddEvents() {
	sve.EditorEvents()
}

// InitScale ensures that Scale is initialized and non-zero
func (sve *Editor) InitScale() {
	if sve.Scale == 0 {
		mvp := sve.Sc
		if mvp != nil {
			sve.Scale = sve.ParentRenderWin().LogicalDPI() / 96.0
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

func (sve *Editor) Render(sc *Scene) {
	wi := sve.This().(Widget)
	if sve.PushBounds(sc) {
		// rs := &sve.Render
		wi.AddEvents()
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
