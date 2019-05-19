// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
)

// todo: for style
// Align = layouts
// Flex -- flexbox -- https://www.w3schools.com/css/css3_flexbox.asp -- key to look at further for layout ideas
// as is Position -- absolute, sticky, etc
// Resize: user-resizability
// z-index

// CSS vs. Layout alignment
//
// CSS has align-self, align-items (for a container, provides a default for
// items) and align-content which only applies to lines in a flex layout (akin
// to a flow layout) -- there is a presumed horizontal aspect to these, except
// align-content, so they are subsumed in the AlignH parameter in this style.
// Vertical-align works as expected, and Text.Align uses left/center/right
//
// LayoutHoriz, Vert both allow explicit Top/Left Center/Middle, Right/Bottom
// alignment along with Justify and SpaceAround -- they use IsAlign functions

// IMPORTANT: any changes here must be updated in stylefuncs.go StyleLayoutFuncs

// LayoutStyle contains style preferences on the layout of the element.
type LayoutStyle struct {
	ZIndex         int         `xml:"z-index" desc:"prop: z-index = ordering factor for rendering depth -- lower numbers rendered first -- sort children according to this factor"`
	AlignH         Align       `xml:"horizontal-align" desc:"prop: horizontal-align = horizontal alignment -- for widget layouts -- not a standard css property"`
	AlignV         Align       `xml:"vertical-align" desc:"prop: vertical-align = vertical alignment -- for widget layouts -- not a standard css property"`
	PosX           units.Value `xml:"x" desc:"prop: x = horizontal position -- often superseded by layout but otherwise used"`
	PosY           units.Value `xml:"y" desc:"prop: y = vertical position -- often superseded by layout but otherwise used"`
	Width          units.Value `xml:"width" desc:"prop: width = specified size of element -- 0 if not specified"`
	Height         units.Value `xml:"height" desc:"prop: height = specified size of element -- 0 if not specified"`
	MaxWidth       units.Value `xml:"max-width" desc:"prop: max-width = specified maximum size of element -- 0  means just use other values, negative means stretch"`
	MaxHeight      units.Value `xml:"max-height" desc:"prop: max-height = specified maximum size of element -- 0 means just use other values, negative means stretch"`
	MinWidth       units.Value `xml:"min-width" desc:"prop: min-width = specified minimum size of element -- 0 if not specified"`
	MinHeight      units.Value `xml:"min-height" desc:"prop: min-height = specified minimum size of element -- 0 if not specified"`
	Margin         units.Value `xml:"margin" desc:"prop: margin = outer-most transparent space around box element -- todo: can be specified per side"`
	Padding        units.Value `xml:"padding" desc:"prop: padding = transparent space around central content of box -- todo: if 4 values it is top, right, bottom, left; 3 is top, right&left, bottom; 2 is top & bottom, right and left"`
	Overflow       Overflow    `xml:"overflow" desc:"prop: overflow = what to do with content that overflows -- default is Auto add of scrollbars as needed -- todo: can have separate -x -y values"`
	Columns        int         `xml:"columns" alt:"grid-cols" desc:"prop: columns = number of columns to use in a grid layout -- used as a constraint in layout if individual elements do not specify their row, column positions"`
	Row            int         `xml:"row" desc:"prop: row = specifies the row that this element should appear within a grid layout"`
	Col            int         `xml:"col" desc:"prop: col = specifies the column that this element should appear within a grid layout"`
	RowSpan        int         `xml:"row-span" desc:"prop: row-span = specifies the number of sequential rows that this element should occupy within a grid layout (todo: not currently supported)"`
	ColSpan        int         `xml:"col-span" desc:"prop: col-span = specifies the number of sequential columns that this element should occupy within a grid layout"`
	ScrollBarWidth units.Value `xml:"scrollbar-width" desc:"prop: scrollbar-width = width of a layout scrollbar"`
}

func (ls *LayoutStyle) Defaults() {
	ls.AlignV = AlignMiddle
	ls.MinWidth.Set(2.0, units.Px)
	ls.MinHeight.Set(2.0, units.Px)
	ls.ScrollBarWidth.Set(16.0, units.Px)
}

func (ls *LayoutStyle) SetStylePost(props ki.Props) {
}

// return the alignment for given dimension
func (ls *LayoutStyle) AlignDim(d Dims2D) Align {
	switch d {
	case X:
		return ls.AlignH
	default:
		return ls.AlignV
	}
}

// position settings, in dots
func (ls *LayoutStyle) PosDots() Vec2D {
	return NewVec2D(ls.PosX.Dots, ls.PosY.Dots)
}

// size settings, in dots
func (ls *LayoutStyle) SizeDots() Vec2D {
	return NewVec2D(ls.Width.Dots, ls.Height.Dots)
}

// size max settings, in dots
func (ls *LayoutStyle) MaxSizeDots() Vec2D {
	return NewVec2D(ls.MaxWidth.Dots, ls.MaxHeight.Dots)
}

// size min settings, in dots
func (ls *LayoutStyle) MinSizeDots() Vec2D {
	return NewVec2D(ls.MinWidth.Dots, ls.MinHeight.Dots)
}

// Align has all different types of alignment -- only some are applicable to
// different contexts, but there is also so much overlap that it makes sense
// to have them all in one list -- some are not standard CSS and used by
// layout
type Align int32

const (
	AlignLeft Align = iota
	AlignTop
	AlignCenter
	// middle = vertical version of center
	AlignMiddle
	AlignRight
	AlignBottom
	AlignBaseline
	// same as CSS space-between
	AlignJustify
	AlignSpaceAround
	AlignFlexStart
	AlignFlexEnd
	AlignTextTop
	AlignTextBottom
	// align to subscript
	AlignSub
	// align to superscript
	AlignSuper
	AlignN
)

//go:generate stringer -type=Align

var KiT_Align = kit.Enums.AddEnumAltLower(AlignN, false, StylePropProps, "Align")

func (ev Align) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Align) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// is this a generalized alignment to start of container?
func IsAlignStart(a Align) bool {
	return (a == AlignLeft || a == AlignTop || a == AlignFlexStart || a == AlignTextTop)
}

// is this a generalized alignment to middle of container?
func IsAlignMiddle(a Align) bool {
	return (a == AlignCenter || a == AlignMiddle)
}

// is this a generalized alignment to end of container?
func IsAlignEnd(a Align) bool {
	return (a == AlignRight || a == AlignBottom || a == AlignFlexEnd || a == AlignTextBottom)
}

// overflow type -- determines what happens when there is too much stuff in a layout
type Overflow int32

const (
	// automatically add scrollbars as needed -- this is pretty much the only sensible option, and is the default here, but Visible is default in html
	OverflowAuto Overflow = iota
	// pretty much the same as auto -- we treat it as such
	OverflowScroll
	// make the overflow visible -- this is generally unsafe and not very feasible and will be ignored as long as possible -- currently falls back on auto, but could go to Hidden if that works better overall
	OverflowVisible
	// hide the overflow and don't present scrollbars (supported)
	OverflowHidden
	OverflowN
)

var KiT_Overflow = kit.Enums.AddEnumAltLower(OverflowN, false, StylePropProps, "Overflow")

func (ev Overflow) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Overflow) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

//go:generate stringer -type=Overflow

////////////////////////////////////////////////////////////////////////////////////////
// Layout Data for actually computing the layout

// SizePrefs represents size preferences
type SizePrefs struct {
	Need Vec2D `desc:"minimum size needed -- set to at least computed allocsize"`
	Pref Vec2D `desc:"preferred size -- start here for layout"`
	Max  Vec2D `desc:"maximum size -- will not be greater than this -- 0 = no constraint, neg = stretch"`
}

// return true if Max < 0 meaning can stretch infinitely along given dimension
func (sp SizePrefs) HasMaxStretch(d Dims2D) bool {
	return (sp.Max.Dim(d) < 0.0)
}

// return true if Pref > Need meaning can stretch more along given dimension
func (sp SizePrefs) CanStretchNeed(d Dims2D) bool {
	return (sp.Pref.Dim(d) > sp.Need.Dim(d))
}

// // 2D margins
// type Margins struct {
// 	left, right, top, bottom float32
// }

// // set a single margin for all items
// func (m *Margins) SetMargin(marg float32) {
// 	m.left = marg
// 	m.right = marg
// 	m.top = marg
// 	m.bottom = marg
// }

// StyleFromProps styles Layout-specific fields from ki.Prop properties
func (ly *Layout) StyleFromProps(par *Layout, props ki.Props, vp *Viewport2D) {
	pr := prof.Start("LayoutFromProps")
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		switch key {
		case "lay":
			if inh, init := StyleInhInit(val, par); inh || init {
				if inh {
					ly.Lay = par.Lay
				} else if init {
					ly.Lay = LayoutHoriz
				}
				return
			}
			switch vt := val.(type) {
			case string:
				ly.Lay.FromString(vt)
			case Layouts:
				ly.Lay = vt
			default:
				if iv, ok := kit.ToInt(val); ok {
					ly.Lay = Layouts(iv)
				} else {
					StyleSetError(key, val)
				}
			}
		case "spacing":
			if inh, init := StyleInhInit(val, par); inh || init {
				if inh {
					ly.Spacing = par.Spacing
				} else if init {
					ly.Spacing.Val = 0
				}
				return
			}
			ly.Spacing.SetIFace(val, key)
		}
	}
	pr.End()
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ly *Layout) StyleToDots(uc *units.Context) {
	ly.Spacing.ToDots(uc)
}
