// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

//go:generate core generate

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
type Types int32 //enums:enum

const (
	// zero value is an unknown type
	UnknownType Types = iota

	// MouseDown happens when a mouse button is pressed down.
	// See MouseButton() for which.
	// See Click for a synthetic event representing a MouseDown
	// followed by MouseUp on the same element with Left (primary)
	// mouse button. Often that is the most useful.
	MouseDown

	// MouseUp happens when a mouse button is released.
	// See MouseButton() for which.
	MouseUp

	// MouseMove is always sent when the mouse is moving but no
	// button is down, even if there might be other higher-level events too.
	// These can be numerous and thus it is typically more efficient
	// to listen to other events derived from this.
	// Not unique, and Prev position is updated during compression.
	MouseMove

	// MouseDrag is always sent when the mouse is moving and there
	// is a button down, even if there might be other higher-level
	// events too.
	// The start pos indicates where (and when) button first was pressed.
	// Not unique and Prev position is updated during compression.
	MouseDrag

	// Click represents a MouseDown followed by MouseUp
	// in sequence on the same element, with the Left (primary) button.
	// This is the typical event for most basic user interaction.
	Click

	// DoubleClick represents two Click events in a row in rapid
	// succession.
	DoubleClick

	// TripleClick represents three Click events in a row in rapid
	// succession.
	TripleClick

	// ContextMenu represents a MouseDown/Up event with the
	// Right mouse button (which is also activated by
	// Control key + Left Click).
	ContextMenu

	// LongPressStart is when the mouse has been relatively stable
	// after MouseDown on an element for a minimum duration (500 msec default).
	LongPressStart

	// LongPressEnd is sent after LongPressStart when the mouse has
	// gone up, moved sufficiently, left the current element,
	// or another input event has happened.
	LongPressEnd

	// MouseEnter is when the mouse enters the bounding box
	// of a new element.  It is used for setting the Hover state,
	// and can trigger cursor changes.
	// See DragEnter for alternative case during Drag events.
	MouseEnter

	// MouseLeave is when the mouse leaves the bounding box
	// of an element, that previously had a MouseEnter event.
	// Given that elements can have overlapping bounding boxes
	// (e.g., child elements within a container), it is not the case
	// that a MouseEnter on a child triggers a MouseLeave on
	// surrounding containers.
	// See DragLeave for alternative case during Drag events.
	MouseLeave

	// LongHoverStart is when the mouse has been relatively stable
	// after MouseEnter on an element for a minimum duration
	// (500 msec default).
	// This triggers the LongHover state typically used for Tooltips.
	LongHoverStart

	// LongHoverEnd is after LongHoverStart when the mouse has
	// moved sufficiently, left the current element,
	// or another input event has happened,
	// thereby terminating the LongHover state.
	LongHoverEnd

	// DragStart is at the start of a drag-n-drop event sequence, when
	// a Draggable element is Active and a sufficient distance of
	// MouseDrag events has occurred to engage the DragStart event.
	DragStart

	// DragMove is for a MouseDrag event during the drag-n-drop sequence.
	// Usually don't need to listen to this one.  MouseDrag is also sent.
	DragMove

	// DragEnter is like MouseEnter but after a DragStart during a
	// drag-n-drop sequence.  MouseEnter is not sent in this case.
	DragEnter

	// DragLeave is like MouseLeave but after a DragStart during a
	// drag-n-drop sequence.  MouseLeave is not sent in this case.
	DragLeave

	// Drop is sent when an item being Dragged is dropped on top of a
	// target element. The event struct should be DragDrop.
	Drop

	// DropDeleteSource is sent to the source Drag element if the
	// Drag-n-Drop event is a Move type, which requires deleting
	// the source element.  The event struct should be DragDrop.
	DropDeleteSource

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
	// The [MouseScroll.Delta] on scroll events is always in real pixel/dot units;
	// low-level sources may be in lines or pages, but we normalize everything
	// to real pixels/dots.
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

	// Select is sent for any direction of selection change
	// on (or within if relevant) a Selectable element.
	// Typically need to query the element(s) to determine current
	// selection state.
	Select

	// Focus is sent when a Focusable element receives keyboard focus (ie: by tabbing).
	Focus

	// FocusLost is sent when a Focusable element loses keyboard focus.
	FocusLost

	// Attend is sent when a Pressable element is programmatically set
	// as Attended through an event. Typically the Attended state is engaged
	// by clicking. Attention is like Focus, in that there is only 1 element
	// at a time in the Attended state, but it does not direct keyboard input.
	// The primary effect of attention is on scrolling events via
	// [abilities.ScrollableUnattended].
	Attend

	// AttendLost is sent when a different Pressable element is Attended.
	AttendLost

	// Change is when a value represented by the element has been changed
	// by the user and committed (for example, someone has typed text in a
	// textfield and then pressed enter). This is *not* triggered when
	// the value has not been committed; see [Input] for that.
	// This is for Editable, Checkable, and Slidable items.
	Change

	// Input is when a value represented by the element has changed, but
	// has not necessarily been committed (for example, this triggers each
	// time someone presses a key in a text field). This *is* triggered when
	// the value has not been committed; see [Change] for a version that only
	// occurs when the value is committed.
	// This is for Editable, Checkable, and Slidable items.
	Input

	// Show is sent to widgets when their Scene is first shown to the user
	// in its final form, and whenever a major content managing widget
	// (e.g., [core.Tabs], [core.Pages]) shows a new tab/page/element (via
	// [core.WidgetBase.Shown] or DeferShown). This can be used for updates
	// that depend on other elements, or relatively expensive updates that
	// should be only done when actually needed "at show time".
	Show

	// Close is sent to widgets when their Scene is being closed. This is an
	// opportunity to save unsaved edits, for example. This is guaranteed to
	// only happen once per widget per Scene.
	Close

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

// IsKey returns true if event type is a Key type
func (tp Types) IsKey() bool {
	return tp >= KeyDown && tp <= KeyChord
}

// IsMouse returns true if event type is a Mouse type
func (tp Types) IsMouse() bool {
	return tp >= MouseDown && tp <= LongHoverEnd
}

// IsTouch returns true if event type is a Touch type
func (tp Types) IsTouch() bool {
	return tp >= TouchStart && tp <= Rotate
}

// IsDrag returns true if event type is a Drag type
func (tp Types) IsDrag() bool {
	return tp >= DragStart && tp <= DragLeave
}

// IsSlide returns true if event type is a Slide type
func (tp Types) IsSlide() bool {
	return tp >= SlideStart && tp <= SlideStop
}

// IsWindow returns true if event type is a Window type
func (tp Types) IsWindow() bool {
	return tp >= Window && tp <= WindowPaint
}

// EventFlags encode boolean event properties
type EventFlags int64 //enums:bitflag

const (
	// Handled indicates that the event has been handled
	Handled EventFlags = iota

	// EventUnique indicates that the event is Unique and not
	// to be compressed with like events.
	Unique
)
