// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"cogentcore.org/core/events/key"
)

// SelectModes interprets the modifier keys to determine what type of selection mode to use.
// This is also used for selection actions and has modes not directly activated by
// modifier keys.
type SelectModes int32 //enums:enum

const (
	// SelectOne selects a single item, and is the default when no modifier key
	// is pressed
	SelectOne SelectModes = iota

	// ExtendContinuous, activated by Shift key, extends the selection to
	// select a continuous region of selected items, with no gaps
	ExtendContinuous

	// ExtendOne, activated by Control or Meta / Command, extends the
	// selection by adding the one additional item just clicked on, creating a
	// potentially discontinuous set of selected items
	ExtendOne

	// NoSelect means do not update selection -- this is used programmatically
	// and not available via modifier key
	NoSelect

	// Unselect means unselect items -- this is used programmatically
	// and not available via modifier key -- typically ExtendOne will
	// unselect if already selected
	Unselect

	// SelectQuiet means select without doing other updates or signals -- for
	// bulk updates with a final update at the end -- used programmatically
	SelectQuiet

	// UnselectQuiet means unselect without doing other updates or signals -- for
	// bulk updates with a final update at the end -- used programmatically
	UnselectQuiet
)

// SelectModeBits returns the selection mode based on given modifiers bitflags
func SelectModeBits(mods key.Modifiers) SelectModes {
	if key.HasAnyModifier(mods, key.Shift) {
		return ExtendContinuous
	}
	if key.HasAnyModifier(mods, key.Meta) {
		return ExtendOne
	}
	return SelectOne
}
