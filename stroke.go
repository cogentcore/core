// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"image/color"
	// "log"
)

// end-cap of a line: stroke-linecap property in SVG
type LineCap int

const (
	LineCapButt LineCap = iota
	LineCapRound
	LineCapSquare
	LineCapN
)

//go:generate stringer -type=LineCap

var KiT_LineCap = ki.Enums.AddEnumAltLower(LineCapButt, false, nil, "LineCap", int64(LineCapN))

// the way in which lines are joined together: stroke-linejoin property in SVG
type LineJoin int

const (
	LineJoinMiter     LineJoin = iota
	LineJoinMiterClip          // SVG2 -- not yet supported
	LineJoinRound
	LineJoinBevel
	LineJoinArcs // SVG2 -- not yet supported
	LineJoinN
)

//go:generate stringer -type=LineJoin

var KiT_LineJoin = ki.Enums.AddEnumAltLower(LineJoinMiter, false, nil, "LineJoin", int64(LineJoinN))

// StrokeStyle contains all the properties specific to painting a line -- the svg elements define the corresponding SVG style attributes, which are processed in StrokeStyle
type StrokeStyle struct {
	On         bool        `desc:"is stroke active -- if property is none then false"`
	Color      Color       `xml:"stroke" desc:"default stroke color when such a color is needed -- Server could be anything"`
	Opacity    float64     `xml:"stroke-opacity" desc:"global alpha opacity / transparency factor"`
	Server     PaintServer `desc:"paint server for the stroke -- if solid color, defines the stroke color"`
	Width      units.Value `xml:"stroke-width" desc:"line width"`
	Dashes     []float64   `xml:"stroke-dasharray" desc:"dash pattern"`
	Cap        LineCap     `xml:"stroke-linecap" desc:"how to draw the end cap of lines"`
	Join       LineJoin    `xml:"stroke-linejoin" desc:"how to join line segments"`
	MiterLimit float64     `xml:"stroke-miterlimit,min:"1" desc:"limit of how far to miter -- must be 1 or larger"`
}

// initialize default values for paint stroke
func (ps *StrokeStyle) Defaults() {
	ps.On = false // svg says default is off
	ps.Server = NewSolidcolorPaintServer(color.Black)
	ps.Width.Set(1.0, units.Px)
	ps.Cap = LineCapButt
	ps.Join = LineJoinMiter // Miter not yet supported, but that is the default -- falls back on bevel
	ps.MiterLimit = 1.0
	ps.Opacity = 1.0
}

// need to do some updating after setting the style from user properties
func (ps *StrokeStyle) SetStylePost() {
	if ps.Color.IsNil() {
		ps.On = false
	} else {
		ps.On = true
		// for now -- todo: find a more efficient way of doing this, and only updating when necc
		ps.Server = NewSolidcolorPaintServer(&ps.Color)
		// todo: incorporate opacity
	}
}

func (ps *StrokeStyle) SetColor(cl *Color) {
	if cl == nil || cl.IsNil() {
		ps.On = false
	} else {
		ps.On = true
		ps.Color = *cl
		ps.Server = NewSolidcolorPaintServer(&ps.Color)
	}
}
