// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"image"
	"image/color"
	"log"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/ki/kit"
)

// see io.go for IO input / output methods

// SVG is a viewport for containing SVG drawing objects, correponding to the
// svg tag in html -- it provides its own bitmap for drawing into
type SVG struct {
	gi.Viewport2D
	ViewBox ViewBox  `desc:"viewbox defines the coordinate system for the drawing"`
	Norm    bool     `desc:"prop: norm = install a transform that renormalizes so that the specified ViewBox exactly fits within the allocated SVG size"`
	InvertY bool     `desc:"prop: invert-y = when doing Norm transform, also flip the Y axis so that the smallest Y value is at the bottom of the SVG box, instead of being at the top as it is by default"`
	Pnt     gi.Paint `json:"-" xml:"-" desc:"paint styles -- inherited by nodes"`
	Defs    Group    `desc:"all defs defined elements go here (gradients, symbols, etc)"`
	Title   string   `xml:"title" desc:"the title of the svg"`
	Desc    string   `xml:"desc" desc:"the description of the svg"`
}

var KiT_SVG = kit.Types.AddType(&SVG{}, nil)

// Paint satisfies the painter interface
func (g *SVG) Paint() *gi.Paint {
	return &g.Pnt
}

// DeleteAll deletes any existing elements in this svg
func (svg *SVG) DeleteAll() {
	updt := svg.UpdateStart()
	svg.DeleteChildren(true)
	svg.ViewBox.Defaults()
	svg.Pnt.Defaults()
	svg.Defs.DeleteChildren(true)
	svg.Title = ""
	svg.Desc = ""
	svg.UpdateEnd(updt)
}

// SetNormXForm sets a scaling transform to make the entire viewbox to fit the viewport
func (svg *SVG) SetNormXForm() {
	pc := &svg.Pnt
	pc.XForm = gi.Identity2D()
	if svg.ViewBox.Size != gi.Vec2DZero {
		// todo: deal with all the other options!
		vpsX := float32(svg.Geom.Size.X) / svg.ViewBox.Size.X
		vpsY := float32(svg.Geom.Size.Y) / svg.ViewBox.Size.Y
		if svg.InvertY {
			vpsY *= -1
		}
		svg.Pnt.XForm = svg.Pnt.XForm.Scale(vpsX, vpsY).Translate(-svg.ViewBox.Min.X, -svg.ViewBox.Min.Y)
		if svg.InvertY {
			svg.Pnt.XForm.Y0 = -svg.Pnt.XForm.Y0
		}
	}
}

// SetDPIXForm sets a scaling transform to compensate for the dpi -- svg
// rendering is done within a 96 DPI context
func (svg *SVG) SetDPIXForm() {
	pc := &svg.Pnt
	dpisc := svg.Viewport.Win.LogicalDPI() / 96.0
	pc.XForm = gi.Scale2D(dpisc, dpisc)
}

func (svg *SVG) Init2D() {
	svg.Viewport2D.Init2D()
	svg.SetFlag(int(gi.VpFlagSVG)) // we are an svg type
	svg.Pnt.Defaults()
	svg.Pnt.FontStyle.BgColor.SetColor(color.White)
}

func (svg *SVG) Size2D(iter int) {
	svg.InitLayout2D()
	if svg.ViewBox.Size != gi.Vec2DZero {
		svg.LayData.AllocSize = svg.ViewBox.Size
	}
	svg.Size2DAddSpace()
}

func (svg *SVG) StyleSVG() {
	svg.Style2DWidget()
	svg.Pnt.Defaults()
	StyleSVG(svg.This().(gi.Node2D))
	svg.Pnt.SetUnitContext(svg.AsViewport2D(), svg.ViewBox.Size) // context is viewbox
}

func (svg *SVG) Style2D() {
	svg.StyleSVG()
	svg.LayData.SetFromStyle(&svg.Sty.Layout) // also does reset
	if nv, ok := svg.Prop("norm"); ok {
		svg.Norm, _ = kit.ToBool(nv)
	}
	if iv, ok := svg.Prop("invert-y"); ok {
		svg.InvertY, _ = kit.ToBool(iv)
	}
}

func (svg *SVG) Layout2D(parBBox image.Rectangle, iter int) bool {
	svg.Layout2DBase(parBBox, true, iter)
	// do not call layout on children -- they don't do it
	// this is too late to affect anything
	// svg.Pnt.SetUnitContext(svg.AsViewport2D(), svg.ViewBox.Size)
	return false
}

func (svg *SVG) Render2D() {
	if svg.PushBounds() {
		rs := &svg.Render
		if svg.Fill {
			svg.FillViewport()
		}
		if svg.Norm {
			svg.SetNormXForm()
		}
		rs.PushXForm(svg.Pnt.XForm)
		svg.Render2DChildren() // we must do children first, then us!
		svg.PopBounds()
		rs.PopXForm()
		svg.RenderViewport2D() // update our parent image
	}
}

func (svg *SVG) FindNamedElement(name string) gi.Node2D {
	name = strings.TrimPrefix(name, "#")
	if name == "" {
		log.Printf("gi.SVG FindNamedElement: name is empty\n")
		return nil
	}
	if svg.Nm == name {
		return svg.This().(gi.Node2D)
	}

	def, ok := svg.Defs.ChildByName(name, 0)
	if ok {
		return def.(gi.Node2D)
	}

	if svg.Par == nil {
		log.Printf("gi.SVG FindNamedElement: could not find name: %v\n", name)
		return nil
	}
	pgi, _ := gi.KiToNode2D(svg.Par)
	if pgi != nil {
		return pgi.FindNamedElement(name)
	}
	log.Printf("gi.SVG FindNamedElement: could not find name: %v\n", name)
	return nil
}
