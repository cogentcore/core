// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"fmt"

	"goki.dev/girl/units"
	"goki.dev/mat32/v2"
)

// todo: for style
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

// IMPORTANT: any changes here must be updated in style_props.go StyleLayoutFuncs

// ScrollBarWidthDefault is the default width of a scrollbar in pixels
var ScrollBarWidthDefault = float32(10)

func (s *Style) LayoutDefaults() {
	s.Gap.Set(units.Em(0.5))
	s.ScrollBarWidth.Dp(ScrollBarWidthDefault)
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (s *Style) LayoutToDots(uc *units.Context) {
	s.Pos.ToDots(uc)
	s.Min.ToDots(uc)
	s.Max.ToDots(uc)
	s.Padding.ToDots(uc)
	s.Margin.ToDots(uc)
	s.Gap.ToDots(uc)
	s.ScrollBarWidth.ToDots(uc)
}

// Align has all different types of alignment -- only some are applicable to
// different contexts, but there is also so much overlap that it makes sense
// to have them all in one list -- some are not standard CSS and used by
// layout
type Align int32 //enums:enum -trim-prefix Align

const (
	// Align items to the start (top, left) of layout
	AlignStart Align = iota

	// Align items to the end (bottom, right) of layout
	AlignEnd

	// Align all items centered around the center of layout space
	AlignCenter

	// Align to text baselines
	AlignBaseline

	// First and last are flush, equal space between remaining items
	AlignSpaceBetween

	// First and last have 1/2 space at edges, full space between remaining items
	AlignSpaceAround

	// Equal space at start, end, and between all items
	AlignSpaceEvenly
)

// overflow type -- determines what happens when there is too much stuff in a layout
type Overflow int32 //enums:enum -trim-prefix Overflow

const (
	// OverflowVisible makes the overflow visible, meaning that the size
	// of the container is always at least the Min size of its contents.
	// No scrollbars are shown.
	OverflowVisible Overflow = iota

	// OverflowHidden hides the overflow and doesn't present scrollbars.
	OverflowHidden

	// OverflowAuto automatically determines if scrollbars should be added to show
	// the overflow.  Scrollbars are added only if the actual content size is greater
	// than the currently available size.
	OverflowAuto

	// OverflowScroll means that scrollbars are always visible,
	// and is otherwise identical to Auto.  However, only during Viewport PrefSize call,
	// the actual content size is used -- otherwise it behaves just like Auto.
	OverflowScroll
)

////////////////////////////////////////////////////////////////////////////////////////
// Layout Data for actually computing the layout

// SizePrefs represents size preferences
type SizePrefs struct { //gti:add

	// minimum size needed -- set to at least computed allocsize
	Need mat32.Vec2

	// preferred size -- start here for layout
	Pref mat32.Vec2

	// maximum size -- will not be greater than this -- 0 = no constraint, neg = stretch
	Max mat32.Vec2
}

func (sp SizePrefs) String() string {
	return fmt.Sprintf("Size Prefs: Need=%s; Pref=%s; Max=%s", sp.Need, sp.Pref, sp.Max)
}

// return true if Max < 0 meaning can stretch infinitely along given dimension
func (sp SizePrefs) HasMaxStretch(d mat32.Dims) bool {
	return (sp.Max.Dim(d) < 0.0)
}

// return true if Pref > Need meaning can stretch more along given dimension
func (sp SizePrefs) CanStretchNeed(d mat32.Dims) bool {
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
