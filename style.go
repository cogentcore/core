// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
// "fmt"
// "github.com/rcoreilly/goki/ki"
// "reflect"
)

////////////////////////////////////////////////////////////////////////////////////////
// Widget Styling

// CSS style reference: https://www.w3schools.com/cssref/default.asp

// styling strategy:
// * indiv objects specify styles using property map -- good b/c it is fully open-ended
// * we process those properties dynamically when rendering (first pass only) into state
//   on objects that can be directly used during rendering
// * good for basic rendering -- lots of additional things that could be extended later..
// * todo: could we generalize this to not have to write the parsing code?  YES need to
//
// SVG Paint inheritance is probably NOT such a good idea for widgets??  fill = background?
// may need to figure that part out a bit more..

// todo: Align = layouts
// Content -- enum of various options
// Items -- similar enum -- combine
// Self "

// todo: Animation

// Bottom = alignment too

// Clear -- no floating elements

// Clip -- clip images

// column- settings -- lots of those

// Flex -- flexbox -- https://www.w3schools.com/css/css3_flexbox.asp -- key to look at further for layout ideas

// FontStyle is in font.go

// TextStyle has text-associated styles in text.go

// List-style for lists

// Object-fit for videos

// Overflow is key for layout: visible, hidden, scroll, auto
// as is Position -- absolute, sticky, etc
// Resize: user-resizability
// vertical-align
// z-index

// visibility -- support more than just hidden

// Transform -- can throw in any 2D or 3D transform!  we support that!  sort of..

// transition -- animation of hover, etc

// style parameters for backgrounds
type BackgroundStyle struct {
	Color Color `xml:"background-color",desc:"background color"`
	// todo: all the properties not yet implemented -- mostly about images
	// Image is like a PaintServer -- includes gradients etc
	// Attachment -- how the image moves
	// Clip -- how to clip the image
	// Origin
	// Position
	// Repeat
	// Size
}

// sides of a box -- some properties can be specified per each side (e.g., border) or not
type BoxSides int32

const (
	BoxTop BoxSides = iota
	BoxRight
	BoxBottom
	BoxLeft
)

//go:generate stringer -type=BoxSides

// how to draw the border
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
)

//go:generate stringer -type=BorderDrawStyle

// style parameters for shadows
type ShadowStyle struct {
	HOffset float64 `xml:"h-offset",desc:"horizontal offset of shadow -- positive = right side, negative = left side"`
	VOffset float64 `xml:"v-offset",desc:"vertical offset of shadow -- positive = below, negative = above"`
	Blur    float64 `xml:"blur",desc:"blur radius -- higher numbers = more blurry"`
	Spread  float64 `xml:"spread",desc:"spread radius -- positive number increases size of shadow, negative descreases size"`
	Color   Color   `xml:"color",desc:"color of the shadow"`
	Inset   bool    `xml:"inset",desc:"shadow is inset within box instead of outset outside of box"`
}

// style parameters for borders
type BorderStyle struct {
	Style  BorderDrawStyle `xml:"style",desc:"how to draw the border"`
	Width  float64         `xml:"width",desc:"width of the border"`
	Radius float64         `xml:"radius",desc:"rounding of the corners"`
	Color  Color           `xml:"color",desc:"color of the border"`
}

// all the CSS-based style elements
type Style struct {
	IsSet         bool            `desc:"has this style been set from object values yet?"`
	Size          Size2D          `xml:"{width,height}",desc:"specified size of element -- 0 if not specified"`
	MaxSize       Size2D          `xml:"{max-width,max-height}",desc:"specified maximum size of element -- 0 if not specified"`
	Minize        Size2D          `xml:"{min-width,min-height}",desc:"specified mimimum size of element -- 0 if not specified"`
	Offsets       []float64       `xml:"{top,right,bottom,left}",desc:"specified offsets for each side"`
	Border        []BorderStyle   `xml:"border",desc:"border around the box element -- can have separate ones for different sides"`
	Margin        float64         `xml:"margin",desc:"outer-most transparent space around box element"`
	Shadow        ShadowStyle     `xml:"box-shadow",desc:"type of shadow to render around box"`
	Padding       []float64       `xml:"padding",desc:"transparent space around central content of box -- if 4 values it is top, right, bottom, left; 3 is top, right&left, bottom; 2 is top & bottom, right and left"`
	Font          FontStyle       `xml:"font",desc:"font parameters"`
	Text          TextStyle       `xml:"font",desc:"font parameters"`
	Color         Color           `xml:"color",desc:"text color"`
	Background    BackgroundStyle `xml:"background-color",desc:"color / fill to fill in background"`
	Opacity       float64         `xml:"opacity",desc:"alpha value to apply to all elements"`
	Outline       BorderStyle     `xml:"outline",desc:"draw an outline around an element -- mostly same styles as border -- default to none"`
	PointerEvents bool            `xml:"pointer-events",desc:"does this element respond to pointer events -- default is true"`
	// todo: also see above for more notes on missing style elements
	// Display    bool            `xml:display",desc:"big enum of how to display item -- controls layout etc"`
}

func (s *Style) Defaults() {
	// mostly all the defaults are 0 initial values, except these..
	s.Opacity = 1.0
	s.Outline.Style = BorderNone
	s.PointerEvents = true
}
