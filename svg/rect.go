// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Rect is a SVG rectangle, optionally with rounded corners
type Rect struct {
	NodeBase
	Pos    mat32.Vec2 `xml:"{x,y}" desc:"position of the top-left of the rectangle"`
	Size   mat32.Vec2 `xml:"{width,height}" desc:"size of the rectangle"`
	Radius mat32.Vec2 `xml:"{rx,ry}" desc:"radii for curved corners, as a proportion of width, height"`
}

var KiT_Rect = kit.Types.AddType(&Rect{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewRect adds a new rectangle to given parent node, with given name, pos, and size.
func AddNewRect(parent ki.Ki, name string, x, y, sx, sy float32) *Rect {
	g := parent.AddNewChild(KiT_Rect, name).(*Rect)
	g.Pos.Set(x, y)
	g.Size.Set(sx, sy)
	return g
}

func (g *Rect) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Rect)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Pos = fr.Pos
	g.Size = fr.Size
	g.Radius = fr.Radius
}

func (g *Rect) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	if g.Radius.X == 0 && g.Radius.Y == 0 {
		pc.DrawRectangle(rs, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y)
	} else {
		// todo: only supports 1 radius right now -- easy to add another
		pc.DrawRoundedRectangle(rs, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y, g.Radius.X)
	}
	pc.FillStrokeClear(rs)
	g.ComputeBBoxSVG()
	g.Render2DChildren()
	rs.PopXForm()
}
