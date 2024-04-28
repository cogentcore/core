// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"

	"cogentcore.org/core/events/key"
)

// events.Key is a low-level immediately generated key event, tracking press
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

func (ev *Key) NeedsFocus() bool {
	return true
}

func (ev *Key) String() string {
	if ev.Typ == KeyChord {
		return fmt.Sprintf("%v{Chord: %v, Rune: %d, Hex: %X, Mods: %v, Time: %v, Handled: %v}", ev.Type(), ev.KeyChord(), ev.Rune, ev.Rune, ev.Mods.ModifiersString(), ev.Time().Format("04:05"), ev.IsHandled())
	}
	return fmt.Sprintf("%v{Code: %v, Mods: %v, Time: %v, Handled: %v}", ev.Type(), ev.Code, ev.Mods.ModifiersString(), ev.Time().Format("04:05"), ev.IsHandled())
}
