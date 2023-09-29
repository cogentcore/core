// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package states

//go:generate enumgen

// States are GUI states of elements that are relevant for styling based on
// [CSS Pseudo-classes](https://developer.mozilla.org/en-US/docs/Web/CSS/Pseudo-classes)
type States int64 //enums:bitflag

const (
	// Disabled elements cannot be interacted with or selected, but do display
	Disabled States = iota

	// ReadOnly elements elements cannot be changed
	ReadOnly

	// Selected elements have been marked for clipboard or other such actions
	Selected

	// Active elements are currently being interacted with,
	// including a button being pressed, an element being dragged or scrolled
	Active

	// Focused elements receive keyboard input
	Focused

	// FocusedWithin elements have a Focused element within them,
	// including self
	FocusedWithin

	// Checked is for check boxes or radio buttons or other similar state
	Checked

	// Hovered indicates that a mouse pointer has entered the space over
	// an element, but it is not Active
	Hovered

	// LongHovered indicates a Hover that persists without significant
	// movement for a minimum period of time (e.g., 500 msec),
	// which typically triggers a tooltip popup
	LongHovered

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
