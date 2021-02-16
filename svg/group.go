// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"image"

	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Group groups together SVG elements -- doesn't do much but provide a
// locus for properties etc
type Group struct {
	NodeBase
}

var KiT_Group = kit.Types.AddType(&Group{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewGroup adds a new group to given parent node, with given name.
func AddNewGroup(parent ki.Ki, name string) *Group {
	return parent.AddNewChild(KiT_Group, name).(*Group)
}

func (g *Group) SVGName() string { return "g" }

func (g *Group) EnforceSVGName() bool { return false }

func (g *Group) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Group)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
}

// BBoxFromChildren sets the Group BBox from children
func BBoxFromChildren(gii gi.Node2D) image.Rectangle {
	bb := image.ZR
	for i, kid := range *gii.Children() {
		_, gi := gi.KiToNode2D(kid)
		if gi != nil {
			if i == 0 {
				bb = gi.BBox
			} else {
				bb = bb.Union(gi.BBox)
			}
		}
	}
	return bb
}

func (g *Group) BBox2D() image.Rectangle {
	bb := BBoxFromChildren(g)
	return bb
}

func (g *Group) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	pc := &g.Pnt
	rs := g.Render()
	rs.PushXFormLock(pc.XForm)

	g.Render2DChildren()
	g.ComputeBBoxSVG() // must come after render

	rs.PopXFormLock()
}
