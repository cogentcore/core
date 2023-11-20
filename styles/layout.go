// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"log/slog"

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

// IMPORTANT: any changes here must be updated in style_props.go StyleLayoutFuncs

// ScrollBarWidthDefault is the default width of a scrollbar in Dp
var ScrollBarWidthDefault = float32(10)

func (s *Style) LayoutDefaults() {
	s.Justify.Defaults()
	s.Align.Defaults()
	s.Gap.Set(units.Em(0.5))
	s.ScrollBarWidth.Dp(ScrollBarWidthDefault)
}

// LayoutHasParSizing returns true if the layout parameters use parent-relative
// sizing units, which requires additional updating during layout
func (s *Style) LayoutHasParSizing() bool {
	if s.Min.X.Un == units.UnitEw || s.Min.X.Un == units.UnitEh ||
		s.Min.Y.Un == units.UnitEw || s.Min.Y.Un == units.UnitEh ||
		s.Max.X.Un == units.UnitEw || s.Max.X.Un == units.UnitEh ||
		s.Max.Y.Un == units.UnitEw || s.Max.Y.Un == units.UnitEh {
		slog.Error("styling error: cannot use Ew or Eh for Min size -- that is self-referential!")
	}

	if s.Min.X.Un == units.UnitPw || s.Min.X.Un == units.UnitPh ||
		s.Min.Y.Un == units.UnitPw || s.Min.Y.Un == units.UnitPh ||
		s.Max.X.Un == units.UnitPw || s.Max.X.Un == units.UnitPh ||
		s.Max.Y.Un == units.UnitPw || s.Max.Y.Un == units.UnitPh {
		return true
	}
	return false
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

// AlignPos returns the position offset based on Align.X,Y settings
// for given inner-sized box within given outer-sized container box.
func AlignPos(align Aligns, inner, outer float32) float32 {
	extra := outer - inner
	var pos float32
	if extra > 0 {
		pos += AlignFactor(align) * extra
	}
	return mat32.Floor(pos)
}

/////////////////////////////////////////////////////////////////

// Direction specifies which way items are laid out.
type Directions int32 //enums:enum

const (
	Row Directions = iota
	Column
)

func (d Directions) Dim() mat32.Dims {
	return mat32.Dims(d)
}

// Displays determines how items are displayed
type Displays int32 //enums:enum -trim-prefix Display

const (
	// Flex is the default layout model, based on a simplified version of the
	// CSS flex layout: uses MainAxis to specify the direction, Wrap for
	// wrapping of elements, and Min, Max, and Grow values on elements to
	// determine sizing.
	Flex Displays = iota

	// Stacked is a stack of elements, with one on top that is visible
	Stacked

	// Grid is the X, Y grid layout, with Columns specifying the number
	// of elements in the X axis.
	Grid

	// NoLayout means that no automatic layout will be applied to elements,
	// which can then be managed via custom code.
	NoLayout

	// None means the item is not displayed: sets the Invisible state
	DisplayNone
)

// Aligns has all different types of alignment and justification.
type Aligns int32 //enums:enum

const (
	// Auto means the item uses the container's AlignItems value
	Auto Aligns = iota

	// Align items to the start (top, left) of layout
	Start

	// Align items to the end (bottom, right) of layout
	End

	// Align items centered
	Center

	// Align to text baselines
	Baseline

	// First and last are flush, equal space between remaining items
	SpaceBetween

	// First and last have 1/2 space at edges, full space between remaining items
	SpaceAround

	// Equal space at start, end, and between all items
	SpaceEvenly
)

func AlignFactor(al Aligns) float32 {
	switch al {
	case Start:
		return 0
	case End:
		return 1
	case Center:
		return 0.5
	}
	return 0
}

// AlignSet specifies the 3 levels of Justify or Align: Content, Items, and Self
type AlignSet struct {
	// Content specifies the distribution of the entire collection of items within
	// any larger amount of space allocated to the container.  By contrast, Items
	// and Self specify distribution within the individual element's allocated space.
	Content Aligns

	// Items specifies the distribution within the individual element's allocated space,
	// as a default for all items within a collection.
	Items Aligns

	// Self specifies the distribution within the individual element's allocated space,
	// for this specific item.  Auto defaults to containers Items setting.
	Self Aligns
}

func (as *AlignSet) Defaults() {
	as.Content = Start
	as.Items = Start
	as.Self = Auto
}

// ItemAlign returns the effective Aligns value between parent Items and Self
func ItemAlign(parItems, self Aligns) Aligns {
	if self == Auto {
		return parItems
	}
	return self
}

// overflow type -- determines what happens when there is too much stuff in a layout
type Overflows int32 //enums:enum -trim-prefix Overflow

const (
	// OverflowVisible makes the overflow visible, meaning that the size
	// of the container is always at least the Min size of its contents.
	// No scrollbars are shown.
	OverflowVisible Overflows = iota

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
