// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package states

//go:generate enumgen

import "goki.dev/enums"

// States are GUI states of elements that are relevant for styling based on
// [CSS Pseudo-classes](https://developer.mozilla.org/en-US/docs/Web/CSS/Pseudo-classes)
type States int64 //enums:bitflag

const (
	// Disabled elements cannot be interacted with or selected, but do display.
	Disabled States = iota

	// ReadOnly elements cannot be changed, but can be selected.
	ReadOnly

	// Selected elements have been marked for clipboard or other such actions.
	Selected

	// Active elements are currently being interacted with,
	// usually involving a mouse button being pressed in the element.
	// A text field will be active while being clicked on, and this
	// can also result in a Focused state.
	// If further movement happens, an element can also end up being
	// Dragged or Sliding.
	Active

	// Dragged means this element is currently being dragged
	// by the mouse (i.e., a MouseDown event followed by MouseMove),
	// as part of a drag-n-drop sequence.
	Dragged

	// Sliding means this element is currently being manipulated
	// via mouse to change the slider state, which will continue
	// until the mouse is released, even if it goes off the element.
	// It should also still be Active.
	Sliding

	// Scrolled means this element is currently being scrolled.
	Scrolled

	// Focused elements receive keyboard input.
	Focused

	// FocusedWithin elements have a Focused element within them,
	// including self.
	FocusedWithin

	// Checked is for check boxes or radio buttons or other similar state.
	Checked

	// Hovered indicates that a mouse pointer has entered the space over
	// an element, but it is not Active (nor DragHovered).
	Hovered

	// LongHovered indicates a Hover that persists without significant
	// movement for a minimum period of time (e.g., 500 msec),
	// which typically triggers a tooltip popup.
	LongHovered

	// DragHovered indicates that a mouse pointer has entered the space over
	// an element, during a drag-n-drop sequence.  This makes it a candidate
	// for a potential drop target.  See DropOK for state in relation to that.
	DragHovered

	// DropOK indicates that a DragHovered element is OK to receive a Drop
	// from the current Dragged item, subject also to the Droppable ability.
	DropOK

	// Invalid indicates that the element has invalid input and
	// needs to be corrected by the user
	Invalid

	// Required indicates that the element must be set by the user
	Required

	// Blank indicates that the element has yet to be set by user
	Blank

	// Link indicates a URL link that has not been visited yet
	Link

	// Visited indicates a URL link that has been visited
	Visited

	// AnyLink is either Link or Visited
	AnyLink
)

// Is is a shortcut for HasFlag for States
func (st *States) Is(flag enums.BitFlag) bool {
	return st.HasFlag(flag)
}
