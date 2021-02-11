// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"github.com/goki/gi/units"
	"github.com/goki/ki/kit"
)

// note: background-color is in FontStyle as it is needed to make that the
// only style needed for text render styling

// // Background has style parameters for backgrounds
// type Background struct {
// 	// todo: all the properties not yet implemented -- mostly about images
// 	// Image is like a PaintServer -- includes gradients etc
// 	// Attachment -- how the image moves
// 	// Clip -- how to clip the image
// 	// Origin
// 	// Position
// 	// Repeat
// 	// Size
// }

// func (b *Background) Defaults() {
// 	b.Color.SetColor(White)
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

var KiT_BoxSides = kit.Enums.AddEnumAltLower(BoxN, kit.NotBitFlag, StylePropProps, "Box")

func (ev BoxSides) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *BoxSides) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// BorderStyles determines how to draw the border
type BorderStyles int32

const (
	BorderSolid BorderStyles = iota
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

//go:generate stringer -type=BorderStyles

var KiT_BorderStyles = kit.Enums.AddEnumAltLower(BorderN, kit.NotBitFlag, StylePropProps, "Border")

func (ev BorderStyles) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *BorderStyles) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// IMPORTANT: any changes here must be updated in style_props.go StyleBorderFuncs

// Border contains style parameters for borders
type Border struct {
	Style  BorderStyles `xml:"style" desc:"prop: border-style = how to draw the border"`
	Width  units.Value  `xml:"width" desc:"prop: border-width = width of the border"`
	Radius units.Value  `xml:"radius" desc:"prop: border-radius = rounding of the corners"`
	Color  Color        `xml:"color" desc:"prop: border-color = color of the border"`
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (bs *Border) ToDots(uc *units.Context) {
	bs.Width.ToDots(uc)
	bs.Radius.ToDots(uc)
}

// IMPORTANT: any changes here must be updated in style_props.go StyleShadowFuncs

// style parameters for shadows
type Shadow struct {
	HOffset units.Value `xml:".h-offset" desc:"prop: .h-offset = horizontal offset of shadow -- positive = right side, negative = left side"`
	VOffset units.Value `xml:".v-offset" desc:"prop: .v-offset = vertical offset of shadow -- positive = below, negative = above"`
	Blur    units.Value `xml:".blur" desc:"prop: .blur = blur radius -- higher numbers = more blurry"`
	Spread  units.Value `xml:".spread" desc:"prop: .spread = spread radius -- positive number increases size of shadow, negative decreases size"`
	Color   Color       `xml:".color" desc:"prop: .color = color of the shadow"`
	Inset   bool        `xml:".inset" desc:"prop: .inset = shadow is inset within box instead of outset outside of box"`
}

func (s *Shadow) HasShadow() bool {
	return (s.HOffset.Dots > 0 || s.VOffset.Dots > 0)
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (s *Shadow) ToDots(uc *units.Context) {
	s.HOffset.ToDots(uc)
	s.VOffset.ToDots(uc)
	s.Blur.ToDots(uc)
	s.Spread.ToDots(uc)
}
