// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
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

// IMPORTANT: any changes here must be updated in style_properties.go StyleLayoutFuncs

// DefaultScrollbarWidth is the default [Style.ScrollbarWidth].
var DefaultScrollbarWidth = units.Dp(10)

func (s *Style) LayoutDefaults() {
	s.Justify.Defaults()
	s.Align.Defaults()
	s.Gap.Set(units.Em(0.5))
	s.ScrollbarWidth = DefaultScrollbarWidth
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (s *Style) LayoutToDots(uc *units.Context) {
	s.Pos.ToDots(uc)
	s.Min.ToDots(uc)
	s.Max.ToDots(uc)
	s.Padding.ToDots(uc)
	s.Margin.ToDots(uc)
	s.Gap.ToDots(uc)
	s.ScrollbarWidth.ToDots(uc)

	// max must be at least as much as min
	if s.Max.X.Dots > 0 {
		s.Max.X.Dots = max(s.Max.X.Dots, s.Min.X.Dots)
	}
	if s.Max.Y.Dots > 0 {
		s.Max.Y.Dots = max(s.Max.Y.Dots, s.Min.Y.Dots)
	}
}

// AlignPos returns the position offset based on Align.X,Y settings
// for given inner-sized box within given outer-sized container box.
func AlignPos(align Aligns, inner, outer float32) float32 {
	extra := outer - inner
	var pos float32
	if extra > 0 {
		pos += AlignFactor(align) * extra
	}
	return math32.Floor(pos)
}

/////////////////////////////////////////////////////////////////

// Direction specifies the way in which elements are laid out, or
// the dimension on which an element is longer / travels in.
type Directions int32 //enums:enum -transform kebab

const (
	// Row indicates that elements are laid out in a row
	// or that an element is longer / travels in the x dimension.
	Row Directions = iota

	// Column indicates that elements are laid out in a column
	// or that an element is longer / travels in the y dimension.
	Column
)

// Dim returns the corresponding dimension for the direction.
func (d Directions) Dim() math32.Dims {
	return math32.Dims(d)
}

// Other returns the opposite (other) direction.
func (d Directions) Other() Directions {
	if d == Row {
		return Column
	}
	return Row
}

// Displays determines how items are displayed.
type Displays int32 //enums:enum -trim-prefix Display -transform kebab

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

	// Custom means that no automatic layout will be applied to elements,
	// which can then be managed via custom code by setting the [Style.Pos] position.
	Custom

	// None means the item is not displayed: sets the Invisible state
	DisplayNone
)

// Aligns has all different types of alignment and justification.
type Aligns int32 //enums:enum -transform kebab

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
type AlignSet struct { //types:add
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
type Overflows int32 //enums:enum -trim-prefix Overflow -transform kebab

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
