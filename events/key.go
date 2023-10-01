// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"

	"goki.dev/goosi/events/key"
)

// events.Key is a low-level immediately-generated key event, tracking press
// and release of keys -- suitable for fine-grained tracking of key events --
// see also events.Key for events that are generated only on key press,
// and that include the full chord information about all the modifier keys
// that were present when a non-modifier key was released
type Key struct {
	Base
}

func NewKey(typ Types, rn rune, code key.Codes, mods key.Modifiers) *Key {
	ev := &Key{}
	ev.Typ = typ
	ev.SetUnique()
	ev.Rune = rn
	ev.Code = code
	ev.Mods = mods
	return ev
}

func (ev *Key) HasPos() bool {
	return false
}

func (ev *Key) String() string {
	if ev.Rune >= 0 {
		return fmt.Sprintf("%v{Chord: %v, Rune: %d, Hex: %X, Mods: %v, Time: %v}", ev.Type(), ev.KeyChord(), ev.Rune, ev.Rune, key.ModsString(ev.Mods), ev.Time())
	}
	return fmt.Sprintf("%v{Code: %v, Mods: %v, Time: %v}", ev.Type(), ev.Code, key.ModsString(ev.Mods), ev.Time())
}
