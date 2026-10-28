// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"strings"
	"unicode"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
)

// The keys of a terminal go to the program running in it, including all
// of the Control keys, which are the ASCII control characters that
// programs use (Control+C to interrupt, Control+N for the next history
// line, and so on). The [keymap] bindings of the GUI are reached in two
// ways, as in other terminals:
//   - Chords with Meta (Command on macOS) never go to the program, so
//     they have their usual functions.
//   - On other platforms, where the bindings use Control for Command,
//     the binding Control+X (for a letter X) is reached with
//     Control+Shift+X, which sends the same control character as
//     Control+X, so the program does not lose any keys.
//
// [KeyFunction] and [ShortcutChord] apply these rules, so that the
// terminal and apps that use it follow the user's key bindings.

// KeyFunction returns the key function of the given key event in
// a terminal, or [keymap.None] if the key goes to the program.
func KeyFunction(e events.Event) keymap.Functions {
	return keyFunctionOf(e.KeyChord())
}

// keyFunctionOf returns the key function of the given key chord in
// a terminal: see [KeyFunction].
func keyFunctionOf(ch key.Chord) keymap.Functions {
	mods, k := key.ModifiersFromString(string(ch))
	if mods.HasFlag(key.Meta) {
		return keymap.Of(ch)
	}
	if isMacOS() || !mods.HasFlag(key.Control) || !mods.HasFlag(key.Shift) || !isLetter(k) {
		return keymap.None
	}
	mods.SetFlag(false, key.Shift)
	return keymap.Of(key.Chord(mods.ModifiersString() + k))
}

// ShortcutChord returns the key chords of the given key function that
// work in a terminal, for showing as a shortcut (see [KeyFunction]):
// its bindings with Meta, and elsewhere than macOS, those with Control
// and a letter, with Shift added. It is "" if there are none.
func ShortcutChord(kf keymap.Functions) key.Chord {
	var chords []string
	for _, ch := range strings.Split(string(kf.Chord()), "\n") {
		mods, k := key.ModifiersFromString(ch)
		switch {
		case mods.HasFlag(key.Meta):
			chords = append(chords, ch)
		case !isMacOS() && mods.HasFlag(key.Control) && !mods.HasFlag(key.Shift) && isLetter(k):
			mods.SetFlag(true, key.Shift)
			chords = append(chords, mods.ModifiersString()+k)
		}
	}
	return key.Chord(strings.Join(chords, "\n"))
}

// isLetter returns whether the given key of a chord is a letter.
func isLetter(k string) bool {
	rs := []rune(k)
	return len(rs) == 1 && unicode.IsLetter(rs[0])
}

// isMacOS returns whether the app is running on macOS.
func isMacOS() bool {
	return key.SystemPlatform == "MacOS"
}
