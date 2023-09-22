// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"

	"goki.dev/goosi"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/mouse"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

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

func (g *Editor) CopyFieldsFrom(frm any) {
	fr := frm.(*Editor)
	// g.SVG.CopyFieldsFrom(&fr.SVG)
	g.Trans = fr.Trans
	g.Scale = fr.Scale
	g.SetDragCursor = fr.SetDragCursor
}

// EditorEvents handles svg editing events
func (svg *Editor) EditorEvents() {
	svg.ConnectEvent(goosi.MouseDragEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		ssvg := recv.Embed(TypeEditor).(*Editor)
		if ssvg.IsDragging() {
			if !ssvg.SetDragCursor {
				goosi.TheApp.Cursor(ssvg.ParentWindow().OSWin).Push(cursor.HandOpen)
				ssvg.SetDragCursor = true
			}
			del := me.Where.Sub(me.From)
			ssvg.Trans.X += float32(del.X)
			ssvg.Trans.Y += float32(del.Y)
			ssvg.SetTransform()
			ssvg.SetFullReRender()
			ssvg.UpdateSig()
		} else {
			if ssvg.SetDragCursor {
				goosi.TheApp.Cursor(ssvg.ParentWindow().OSWin).Pop()
				ssvg.SetDragCursor = false
			}
		}

	})
	svg.ConnectEvent(goosi.MouseScrollEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		ssvg := recv.Embed(TypeEditor).(*Editor)
		if ssvg.SetDragCursor {
			goosi.TheApp.Cursor(ssvg.ParentWindow().OSWin).Pop()
			ssvg.SetDragCursor = false
		}
		ssvg.InitScale()
		ssvg.Scale += float32(me.NonZeroDelta(false)) / 20
		if ssvg.Scale <= 0 {
			ssvg.Scale = 0.01
		}
		ssvg.SetTransform()
		ssvg.SetFullReRender()
		ssvg.UpdateSig()
	})
	svg.ConnectEvent(goosi.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		ssvg := recv.Embed(TypeEditor).(*Editor)
		if ssvg.SetDragCursor {
			goosi.TheApp.Cursor(ssvg.ParentWindow().OSWin).Pop()
			ssvg.SetDragCursor = false
		}
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if me.Action == mouse.Release && me.Button == mouse.Right {
			me.SetProcessed()
			// if obj != nil {
			// 	giv.StructViewDialog(ssvg.Viewport, obj, giv.DlgOpts{Title: "SVG Element View"}, nil, nil)
			// }
		}
	})
	svg.ConnectEvent(goosi.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.HoverEvent)
		me.SetProcessed()
		ssvg := recv.Embed(TypeEditor).(*Editor)
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if obj != nil {
			pos := me.Where
			ttxt := fmt.Sprintf("element name: %v -- use right mouse click to edit", obj.Name())
			PopupTooltip(obj.Name(), pos.X, pos.Y, svg.Vp, ttxt)
		}
	})
}

func (svg *Editor) ConnectEvents() {
	svg.EditorEvents()
}

// InitScale ensures that Scale is initialized and non-zero
func (svg *Editor) InitScale() {
	if svg.Scale == 0 {
		mvp := svg.Vp
		if mvp != nil {
			svg.Scale = svg.ParentWindow().LogicalDPI() / 96.0
		} else {
			svg.Scale = 1
		}
	}
}

// SetTransform sets the transform based on Trans and Scale values
func (svg *Editor) SetTransform() {
	svg.InitScale()
	svg.SetProp("transform", fmt.Sprintf("translate(%v,%v) scale(%v,%v)", svg.Trans.X, svg.Trans.Y, svg.Scale, svg.Scale))
}

func (svg *Editor) Render(vp *Viewport) {
	if svg.PushBounds() {
		rs := &svg.Render
		svg.This().(Widget).ConnectEvents()
		if svg.Fill {
			svg.FillViewport()
		}
		if svg.Norm {
			svg.SetNormXForm()
		}
		rs.PushXForm(svg.Pnt.XForm)
		svg.RenderChildren() // we must do children first, then us!
		svg.PopBounds()
		rs.PopXForm()
		// fmt.Printf("geom.bounds: %v  geom: %v\n", svg.Geom.Bounds(), svg.Geom)
		svg.RenderViewport() // update our parent image
	}
}
