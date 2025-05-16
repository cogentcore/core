// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package abilities

//go:generate core generate

import "cogentcore.org/core/enums"

// Abilities represent abilities of GUI elements to take on different States,
// and are aligned with the States flags.  All elements can be disabled.
// These correspond to some of the global attributes in CSS:
// https://developer.mozilla.org/en-US/docs/Web/HTML/Global_attributes
type Abilities int64 //enums:bitflag

const (
	// Selectable means it can be Selected
	Selectable Abilities = iota

	// Activatable means it can be made Active by pressing down on it,
	// which gives it a visible state layer color change.
	// This also implies Clickable, receiving Click events when
	// the user executes a mouse down and up event on the same element.
	Activatable

	// Clickable means it can be Clicked, receiving Click events when
	// the user executes a mouse down and up event on the same element,
	// but otherwise does not change its rendering when pressed
	// (as Activatable does).  Use this for items that are more passively
	// clickable, such as frames or tables, whereas e.g., a Button is
	// Activatable.
	Clickable

	// DoubleClickable indicates that an element does something different
	// when it is clicked on twice in a row.
	DoubleClickable

	// TripleClickable indicates that an element does something different
	// when it is clicked on three times in a row.
	TripleClickable

	// RepeatClickable indicates that an element should receive repeated
	// click events when the pointer is held down on it.
	RepeatClickable

	// LongPressable indicates that an element can be LongPressed.
	LongPressable

	// Draggable means it can be Dragged
	Draggable

	// Droppable means it can receive DragEnter, DragLeave, and Drop events
	// (not specific to current Drag item, just generally).
	Droppable

	// Slideable means it has a slider element that can be dragged
	// to change value.  Cannot be both Draggable and Slideable.
	Slideable

	// Checkable means it can be Checked.
	Checkable

	// Scrollable means it can be Scrolled.
	Scrollable

	// Focusable means it can be Focused: capable of receiving and
	// processing key events directly and typically changing the
	// style when focused to indicate this property to the user.
	Focusable

	// Hoverable means it can be Hovered.
	Hoverable

	// LongHoverable means it can be LongHovered.
	LongHoverable

	// ScrollableUnattended means it can be Scrolled and Slided without
	// Focused or Attended state. This is true by default only for Frames.
	ScrollableUnattended
)

var (
	// Pressable is the list of abilities that makes something Pressable
	Pressable = []Abilities{Selectable, Activatable, DoubleClickable, TripleClickable, Draggable, Slideable, Checkable, Clickable}

	pressableBits = []enums.BitFlag{Selectable, Activatable, DoubleClickable, TripleClickable, Draggable, Slideable, Checkable, Clickable}
)

// Is is a shortcut for HasFlag for Abilities
func (ab *Abilities) Is(flag enums.BitFlag) bool {
	return ab.HasFlag(flag)
}

// IsPressable returns true when an element is Selectable, Activatable,
// DoubleClickable, Draggable, Slideable, or Checkable
func (ab *Abilities) IsPressable() bool {
	return enums.HasAnyFlags((*int64)(ab), pressableBits...)
}

// IsHoverable is true for both Hoverable and LongHoverable
func (ab *Abilities) IsHoverable() bool {
	return ab.HasFlag(Hoverable) || ab.HasFlag(LongHoverable)
}
