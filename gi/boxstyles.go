// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/goki/gi/units"
	"github.com/goki/ki/kit"
)

// note: background-color is in FontStyle as it is needed to make that the
// only style needed for text render styling

// // BackgroundStyle has style parameters for backgrounds
// type BackgroundStyle struct {
// 	// todo: all the properties not yet implemented -- mostly about images
// 	// Image is like a PaintServer -- includes gradients etc
// 	// Attachment -- how the image moves
// 	// Clip -- how to clip the image
// 	// Origin
// 	// Position
// 	// Repeat
// 	// Size
// }

// func (b *BackgroundStyle) Defaults() {
// 	b.Color.SetColor(color.White)
// }

// BoxSides specifies sides of a box -- some properties can be specified per
// each side (e.g., border) or not
type BoxSides int32

const (
	BoxTop BoxSides = iota
	BoxRight
	BoxBottom
	BoxLeft
	BoxN
)

//go:generate stringer -type=BoxSides

var KiT_BoxSides = kit.Enums.AddEnumAltLower(BoxN, false, StylePropProps, "Box")

func (ev BoxSides) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *BoxSides) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// BorderDrawStyle determines how to draw the border
type BorderDrawStyle int32

const (
	BorderSolid BorderDrawStyle = iota
	BorderDotted
	BorderDashed
	BorderDouble
	BorderGroove
	BorderRidge
	BorderInset
	BorderOutset
	BorderNone
	BorderHidden
	BorderN
)

//go:generate stringer -type=BorderDrawStyle

var KiT_BorderDrawStyle = kit.Enums.AddEnumAltLower(BorderN, false, StylePropProps, "Border")

func (ev BorderDrawStyle) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *BorderDrawStyle) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// IMPORTANT: any changes here must be updated in stylefuncs.go StyleBorderFuncs

// BorderStyle contains style parameters for borders
type BorderStyle struct {
	Style  BorderDrawStyle `xml:"style" desc:"prop: border-style = how to draw the border"`
	Width  units.Value     `xml:"width" desc:"prop: border-width = width of the border"`
	Radius units.Value     `xml:"radius" desc:"prop: border-radius = rounding of the corners"`
	Color  Color           `xml:"color" desc:"prop: border-color = color of the border"`
}

// IMPORTANT: any changes here must be updated in stylefuncs.go StyleShadowFuncs

// style parameters for shadows
type ShadowStyle struct {
	HOffset units.Value `xml:".h-offset" desc:"prop: .h-offset = horizontal offset of shadow -- positive = right side, negative = left side"`
	VOffset units.Value `xml:".v-offset" desc:"prop: .v-offset = vertical offset of shadow -- positive = below, negative = above"`
	Blur    units.Value `xml:".blur" desc:"prop: .blur = blur radius -- higher numbers = more blurry"`
	Spread  units.Value `xml:".spread" desc:"prop: .spread = spread radius -- positive number increases size of shadow, negative decreases size"`
	Color   Color       `xml:".color" desc:"prop: .color = color of the shadow"`
	Inset   bool        `xml:".inset" desc:"prop: .inset = shadow is inset within box instead of outset outside of box"`
}

func (s *ShadowStyle) HasShadow() bool {
	return (s.HOffset.Dots > 0 || s.VOffset.Dots > 0)
}
