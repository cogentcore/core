// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package states

import "goki.dev/enums"

// Abilities represent abilities of GUI elements to take on different States,
// and are aligned with the States flags.  All elements can be disabled.
// these correspond to some of the global attributes in CSS:
// [MDN](https://developer.mozilla.org/en-US/docs/Web/HTML/Global_attributes)
type Abilities int64 //enums:bitflag

const (
	// Editable means it can switch between ReadOnly and not
	Editable Abilities = iota

	// Selectable means it can be Selected
	Selectable

	// Activatable means it can be made Active
	Activatable

	// Draggable means it can be Dragged
	Draggable

	// Droppable means it can receive Drop events
	// (not specific to current Drag item, just generally)
	Droppable

	// Slideable means it has a slider element that can be dragged
	// to change value.  Cannot be both Draggable and Slideable.
	Slideable

	// Checkable means it can be Checked
	Checkable

	// Scrollable means it can be Scrolled
	Scrollable

	// Focusable means it can be Focused
	Focusable

	// FocusWithinable means it can be FocusedWithin
	FocusWithinable

	// Hoverable means it can be Hovered
	Hoverable

	// LongHoverable means it can be LongHovered
	LongHoverable
)

// Is is a shortcut for HasFlag for Abilities
func (ab *Abilities) Is(flag enums.BitFlag) bool {
	return ab.HasFlag(flag)
}

// IsPressable is true when an element is Selectable, Activatable, Draggable, Slideable, or Checkable
func (ab *Abilities) IsPressable() bool {
	return ab.HasFlag(Selectable) || ab.HasFlag(Activatable) || ab.HasFlag(Draggable) || ab.HasFlag(Slideable) || ab.HasFlag(Checkable)
}

// IsHoverable is true for both Hoverable and LongHoverable
func (ab *Abilities) IsHoverable() bool {
	return ab.HasFlag(Hoverable) || ab.HasFlag(LongHoverable)
}
