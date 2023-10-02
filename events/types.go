// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

//go:generate enumgen

// Types determines the type of GUI event, and also the
// level at which one can select which events to listen to.
// The type should include both the source / nature of the event
// and the "action" type of the event (e.g., MouseDown, MouseUp
// are separate event types). The standard
// [JavaScript Event](https://developer.mozilla.org/en-US/docs/Web/Events)
// provide the basis for most of the event type names and categories.
// Most events use the same Base type and only need
// to set relevant fields and the type.
// Unless otherwise noted, all events are marked as Unique,
// meaning they are always sent.  Non-Unique events are subject
// to compression, where if the last event added (and not yet
// processed and therefore removed from the queue) is of the same type
// then it is replaced with the new one, instead of adding.
type Types int64 //enums:enum

const (
	// zero value is an unknown type
	UnknownType Types = iota

	// MouseDown happens when a mouse button is pressed down. See Button() for which.
	// See Click for a synthetic event representing a MouseDown followed by MouseUp
	// on the same element -- often that is the most useful.
	MouseDown

	// MouseUp happens when a mouse button is released. See Button() for which.
	MouseUp

	// MouseMove is always sent when the mouse is moving but no button is down,
	// even if there might be other higher-level events too.
	// These can be numerous and thus it is typically more efficient
	// to listen to other events derived from this.
	// Not unique, and Prev position is updated during compression.
	MouseMove

	// MouseDrag is always sent when the mouse is moving and there
	// is a button down, even if there might be other higher-level events too.
	// The start pos indicates where (and when) button first was pressed.
	// Not unique and Prev position is updated during compression.
	MouseDrag

	// Click represents a MouseDown followed by MouseUp in sequence on the
	// same element, with the same button. See Button() for which.
	// This is the typical event for most basic user interaction.
	Click

	// DoubleClick represents two Click events in a row in rapid succession.
	// See Button() for which.
	DoubleClick

	// MouseEnter is when the mouse enters the bounding box of a new element.
	// It is used for setting the Hover state, and can trigger cursor changes.
	// See DragEnter for alternative case during Drag events.
	MouseEnter

	// MouseLeave is when the mouse leaves the bounding box of an element,
	// that previously had a MouseEnter event triggered.
	// Given that elements can have overlapping bounding boxes
	// (e.g., child elements within a container), it is not the case
	// that a MouseEnter on a child triggers a MouseLeave on surrounding
	// containers.
	// See DragLeave for alternative case during Drag events.
	MouseLeave

	// LongHoverStart is when the mouse has been relatively stable after
	// MouseEnter on an element for a minimum duration (500 msec default).
	// This triggers the LongHover state typically used for Tooltips.
	LongHoverStart

	// LongHoverEnd is after LongHoverStart when the mouse has
	// moved sufficiently, left the current element,
	// or another input event has happened,
	// thereby terminating the LongHover state.
	LongHoverEnd

	// DragStart is at the start of a drag-n-drop event sequence, when
	// a Draggable element is Active and a sufficient distance of MouseDrag
	// events has occurred to engage the DragStart event.
	DragStart

	// Drop is the final action of the drag-n-drop sequence, when
	// an item being Dragged is dropped on top of a target element.
	// This is also triggered with a nil target if the Escape key
	// is pressed while dragging.  The target will also be nil if
	// the target does not have the DropOK state active.
	Drop

	// DragMove is for a MouseDrag event during the drag-n-drop sequence.
	// Usually don't need to listen to this one.  MouseDrag is also sent.
	DragMove

	// DragEnter is like MouseEnter but after a DragStart during a
	// drag-n-drop sequence.  MouseEnter is not sent in this case.
	DragEnter

	// DragLeave is like MouseLeave but after a DragStart during a
	// drag-n-drop sequence.  MouseLeave is not sent in this case.
	DragLeave

	// SlideStart is for a Slideable element when Active and a
	// sufficient distance of MouseDrag events has occurred to
	// engage the SlideStart event.  Sets the Sliding state.
	SlideStart

	// SlideMove is for a Slideable element after SlideStart
	// is being dragged via MouseDrag events.
	SlideMove

	// SlideStop is when the mouse button is released on a Slideable
	// element being dragged via MouseDrag events.  This typically
	// also accompanied by a Changed event for the new slider value.
	SlideStop

	// Scroll is for scroll wheel or other scrolling events (gestures).
	// These are not unique and Delta is updated during compression.
	Scroll

	// KeyDown is when a key is pressed down.
	// This provides fine-grained data about each key as it happens.
	// KeyChord is recommended for a more complete Key event.
	KeyDown

	// KeyUp is when a key is released.
	// This provides fine-grained data about each key as it happens.
	// KeyChord is recommended for a more complete Key event.
	KeyUp

	// KeyChord is only generated when a non-modifier key is released,
	// and it also contains a string representation of the full chord,
	// suitable for translation into keyboard commands, emacs-style etc.
	// It can be somewhat delayed relative to the KeyUp.
	KeyChord

	// TouchStart is when a touch event starts, for the low-level touch
	// event processing.  TouchStart also activates MouseDown, Scroll,
	// Magnify, or Rotate events depending on gesture recognition.
	TouchStart

	// TouchEnd is when a touch event ends, for the low-level touch
	// event processing.  TouchEnd also activates MouseUp events
	// depending on gesture recognition.
	TouchEnd

	// TouchMove is when a touch event moves, for the low-level touch
	// event processing.  TouchMove also activates MouseMove, Scroll,
	// Magnify, or Rotate events depending on gesture recognition.
	TouchMove

	// Magnify is a touch-based magnify event (e.g., pinch)
	Magnify

	// Rotate is a touch-based rotate event.
	Rotate

	// Select is sent when a Selectable element is selected.
	Select

	// Deselect is sent when a Selectable element is deselected.
	Deselect

	// Focus is sent when Focsable element receives Focus
	Focus

	// FocusLost is sent when Focsable element loses Focus
	FocusLost

	// Change is when a value represented by the element has changed.
	// This is for Editable, Checkable, Slidable items.
	Change

	// Window reports on changes in the window position,
	// visibility (iconify), focus changes, screen update, and closing.
	// These are only sent once per event (Unique).
	Window

	// WindowResize happens when the window has been resized,
	// which can happen continuously during a user resizing
	// episode.  These are not Unique events, and are compressed
	// to minimize lag.
	WindowResize

	// WindowPaint is sent continuously at FPS frequency
	// (60 frames per second by default) to drive updating check
	// on the window.  It is not unique, will be compressed
	// to keep pace with updating.
	WindowPaint

	// OS is an operating system generated event (app level typically)
	OS

	// OSOpenFiles is an event telling app to open given files
	OSOpenFiles

	// Custom is a user-defined event with a data any field
	Custom
)

// EventFlags encode boolean event properties
type EventFlags int64 //enums:bitflag

const (
	// Handled indicates that the event has been handled
	Handled EventFlags = iota

	// EventUnique indicates that the event is Unique and not
	// to be compressed with like events.
	Unique
)
