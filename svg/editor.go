// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Editor supports editing of SVG elements
type Editor struct {
	SVG
	Trans         gi.Vec2D `desc:"view translation offset (from dragging)"`
	Scale         float32  `desc:"view scaling (from zooming)"`
	SetDragCursor bool     `desc:"has dragging cursor been set yet?"`
}

var KiT_Editor = kit.Types.AddType(&Editor{}, nil)

// AddNewEditor adds a new editor to given parent node, with given name.
func AddNewEditor(parent ki.Ki, name string) *Editor {
	return parent.AddNewChild(KiT_Editor, name).(*Editor)
}

// EditorEvents handles svg editing events
func (svg *Editor) EditorEvents() {
	svg.ConnectEvent(oswin.MouseDragEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		ssvg := recv.Embed(KiT_Editor).(*Editor)
		if ssvg.IsDragging() {
			if !ssvg.SetDragCursor {
				oswin.TheApp.Cursor(ssvg.Viewport.Win.OSWin).Push(cursor.HandOpen)
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
				oswin.TheApp.Cursor(ssvg.Viewport.Win.OSWin).Pop()
				ssvg.SetDragCursor = false
			}
		}

	})
	svg.ConnectEvent(oswin.MouseScrollEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		ssvg := recv.Embed(KiT_Editor).(*Editor)
		if ssvg.SetDragCursor {
			oswin.TheApp.Cursor(ssvg.Viewport.Win.OSWin).Pop()
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
	svg.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		ssvg := recv.Embed(KiT_Editor).(*Editor)
		if ssvg.SetDragCursor {
			oswin.TheApp.Cursor(ssvg.Viewport.Win.OSWin).Pop()
			ssvg.SetDragCursor = false
		}
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if me.Action == mouse.Release && me.Button == mouse.Right {
			me.SetProcessed()
			if obj != nil {
				giv.StructViewDialog(ssvg.Viewport, obj, giv.DlgOpts{Title: "SVG Element View"}, nil, nil)
			}
		}
	})
	svg.ConnectEvent(oswin.MouseHoverEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.HoverEvent)
		me.SetProcessed()
		ssvg := recv.Embed(KiT_Editor).(*Editor)
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if obj != nil {
			pos := me.Where
			ttxt := fmt.Sprintf("element name: %v -- use right mouse click to edit", obj.Name())
			gi.PopupTooltip(obj.Name(), pos.X, pos.Y, svg.Viewport, ttxt)
		}
	})
}

// InitScale ensures that Scale is initialized and non-zero
func (svg *Editor) InitScale() {
	if svg.Scale == 0 {
		if svg.Viewport != nil {
			svg.Scale = svg.Viewport.Win.LogicalDPI() / 96.0
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

func (svg *Editor) Render2D() {
	if svg.PushBounds() {
		rs := &svg.Render
		svg.EditorEvents()
		if svg.Fill {
			svg.FillViewport()
		}
		rs.PushXForm(svg.Pnt.XForm)
		svg.Render2DChildren() // we must do children first, then us!
		svg.PopBounds()
		rs.PopXForm()
		svg.RenderViewport2D() // update our parent image
	}
}
