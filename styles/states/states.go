// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package states

//go:generate core generate

import "cogentcore.org/core/enums"

// States are GUI states of elements that are relevant for styling based on
// CSS pseudo-classes (https://developer.mozilla.org/en-US/docs/Web/CSS/Pseudo-classes).
type States int64 //enums:bitflag

const (
	// Invisible elements are not displayable, and thus do not present
	// a target for GUI events. It is identical to CSS display:none.
	// It is often used for elements such as tabs to hide elements in
	// tabs that are not open. Elements can be made visible by toggling
	// this flag and thus in general should be constructed and styled,
	// but a new layout step must generally be taken after visibility
	// status has changed. See also [cogentcore.org/core/core.WidgetBase.IsDisplayable].
	Invisible States = iota

	// Disabled elements cannot be interacted with or selected,
	// but do display.
	Disabled

	// ReadOnly elements cannot be changed, but can be selected.
	// A text input must not be ReadOnly for entering text.
	// A button can be pressed while ReadOnly -- if not ReadOnly then
	// the label on the button can be edited, for example.
	ReadOnly

	// Selected elements have been marked for clipboard or other such actions.
	Selected

	// Active elements are currently being interacted with,
	// usually involving a mouse button being pressed in the element.
	// A text field will be active while being clicked on, and this
	// can also result in a [Focused] state.
	// If further movement happens, an element can also end up being
	// Dragged or Sliding.
	Active

	// Dragging means this element is currently being dragged
	// by the mouse (i.e., a MouseDown event followed by MouseMove),
	// as part of a drag-n-drop sequence.
	Dragging

	// Sliding means this element is currently being manipulated
	// via mouse to change the slider state, which will continue
	// until the mouse is released, even if it goes off the element.
	// It should also still be [Active].
	Sliding

	// The current Focused element receives keyboard input.
	// Only one element can be Focused at a time.
	Focused

	// Attended is the last Pressable element to be clicked on.
	// Only one element can be Attended at a time.
	// The main effect of Attended is on scrolling events:
	// see [abilities.ScrollableUnattended]
	Attended

	// Checked is for check boxes or radio buttons or other similar state.
	Checked

	// Indeterminate indicates that the true state of an item is unknown.
	// For example, [Checked] state items may be in an uncertain state
	// if they represent other checked items, some of which are checked
	// and some of which are not.
	Indeterminate

	// Hovered indicates that a mouse pointer has entered the space over
	// an element, but it is not [Active] (nor [DragHovered]).
	Hovered

	// LongHovered indicates a Hover event that persists without significant
	// movement for a minimum period of time (e.g., 500 msec),
	// which typically triggers a tooltip popup.
	LongHovered

	// LongPressed indicates a MouseDown event that persists without significant
	// movement for a minimum period of time (e.g., 500 msec),
	// which typically triggers a tooltip and/or context menu popup.
	LongPressed

	// DragHovered indicates that a mouse pointer has entered the space over
	// an element during a drag-n-drop sequence.  This makes it a candidate
	// for a potential drop target.
	DragHovered
)

// Is is a shortcut for HasFlag for States
func (st States) Is(flag enums.BitFlag) bool {
	return st.HasFlag(flag)
}

// StateLayer returns the state layer opacity for the state, appropriate for use
// as the value of [cogentcore.org/core/styles.Style.StateLayer]
func (st States) StateLayer() float32 {
	switch {
	case st.Is(Disabled):
		return 0
	case st.Is(Dragging), st.Is(LongPressed):
		return 0.12
	case st.Is(Active), st.Is(Focused):
		return 0.10
	case st.Is(Hovered), st.Is(DragHovered):
		return 0.08
	default:
		return 0
	}
}
