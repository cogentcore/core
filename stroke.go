// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image/color"
	"log"
	"strconv"
	"strings"

	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// end-cap of a line: stroke-linecap property in SVG
type LineCap int

const (
	LineCapButt LineCap = iota
	LineCapRound
	LineCapSquare
	// rasterx extension
	LineCapCubic
	// rasterx extension
	LineCapQuadratic
	LineCapN
)

//go:generate stringer -type=LineCap

var KiT_LineCap = kit.Enums.AddEnumAltLower(LineCapN, false, StylePropProps, "LineCap")

func (ev LineCap) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *LineCap) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// the way in which lines are joined together: stroke-linejoin property in SVG
type LineJoin int

const (
	LineJoinMiter LineJoin = iota
	LineJoinMiterClip
	LineJoinRound
	LineJoinBevel
	LineJoinArcs
	// rasterx extension
	LineJoinArcsClip
	LineJoinN
)

//go:generate stringer -type=LineJoin

var KiT_LineJoin = kit.Enums.AddEnumAltLower(LineJoinN, false, StylePropProps, "LineJoin")

func (ev LineJoin) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *LineJoin) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// StrokeStyle contains all the properties for painting a line
type StrokeStyle struct {
	On         bool        `desc:"is stroke active -- if property is none then false"`
	Color      ColorSpec   `xml:"stroke" desc:"prop: stroke = stroke color specification"`
	Opacity    float32     `xml:"stroke-opacity" desc:"prop: stroke-opacity = global alpha opacity / transparency factor"`
	Width      units.Value `xml:"stroke-width" desc:"prop: stroke-width = line width"`
	MinWidth   units.Value `xml:"stroke-min-width" desc:"prop: stroke-min-width = minimum line width used for rendering -- if width is > 0, then this is the smallest line width -- this value is NOT subject to transforms so is in absolute dot values, and is ignored if vector-effects non-scaling-stroke is used -- this is an extension of the SVG / CSS standard"`
	Dashes     []float64   `xml:"stroke-dasharray" desc:"prop: stroke-dasharray = dash pattern, in terms of alternating on and off distances -- e.g., [4 4] = 4 pixels on, 4 pixels off.  Currently only supporting raw pixel numbers, but in principle should support units."`
	Cap        LineCap     `xml:"stroke-linecap" desc:"prop: stroke-linecap = how to draw the end cap of lines"`
	Join       LineJoin    `xml:"stroke-linejoin" desc:"prop: stroke-linejoin = how to join line segments"`
	MiterLimit float32     `xml:"stroke-miterlimit" min:"1" desc:"prop: stroke-miterlimit = limit of how far to miter -- must be 1 or larger"`
}

// Defaults initializes default values for paint stroke
func (ps *StrokeStyle) Defaults() {
	ps.On = false // svg says default is off
	ps.SetColor(color.Black)
	ps.Width.Set(1.0, units.Px)
	ps.MinWidth.Set(.5, units.Dot)
	ps.Cap = LineCapButt
	ps.Join = LineJoinMiter // Miter not yet supported, but that is the default -- falls back on bevel
	ps.MiterLimit = 10.0
	ps.Opacity = 1.0
}

// SetStylePost does some updating after setting the style from user properties
func (ps *StrokeStyle) SetStylePost(props ki.Props) {
	if ps.Color.IsNil() {
		ps.On = false
	} else {
		ps.On = true
	}
}

// SetColor sets a solid stroke color -- nil turns off stroking
func (ps *StrokeStyle) SetColor(cl color.Color) {
	if cl == nil {
		ps.On = false
	} else {
		ps.On = true
		ps.Color.Color.SetColor(cl)
		ps.Color.Source = SolidColor
	}
}

// SetColorSpec sets full color spec from source
func (ps *StrokeStyle) SetColorSpec(cl *ColorSpec) {
	if cl == nil {
		ps.On = false
	} else {
		ps.On = true
		ps.Color.CopyFrom(cl)
	}
}

// ParseDashesString gets a dash slice from given string
func ParseDashesString(str string) []float64 {
	if len(str) == 0 || str == "none" {
		return nil
	}
	ds := strings.Split(str, ",")
	dl := make([]float64, len(ds))
	for i, dstr := range ds {
		d, err := strconv.ParseFloat(strings.TrimSpace(dstr), 64)
		if err != nil {
			log.Printf("gi.ParseDashesString parsing error: %v\n", err)
			return nil
		}
		dl[i] = d
	}
	return dl
}
