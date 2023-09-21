// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

// FocusChanges are the kinds of changes that can be reported via
// FocusChanged2D method
type FocusChanges int32 //enums:enum

const (
	// FocusLost means that keyboard focus is on a different widget
	// (typically) and this one lost focus
	FocusLost FocusChanges = iota

	// FocusGot means that this widget just got keyboard focus
	FocusGot

	// FocusInactive means that although this widget retains keyboard focus
	// (nobody else has it), the user has clicked on something else and
	// therefore the focus should be considered inactive (distracted), and any
	// changes should be applied as this other action could result in closing
	// of a dialog etc.  Keyboard events will still be sent to the focus
	// widget, but it is up to the widget if or how to process them (e.g., it
	// could reactivate on its own).
	FocusInactive

	// FocusActive means that the user has moved the mouse back into the
	// focused widget to resume active keyboard focus.
	FocusActive
)
