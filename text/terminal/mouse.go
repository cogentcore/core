// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"

	"cogentcore.org/core/events/key"
	"cogentcore.org/core/text/terminal/vt"
)

// mouseAction is the kind of mouse event to report to the program.
type mouseAction int

const (
	mousePress mouseAction = iota
	mouseRelease
	mouseMotion
)

// Mouse button numbers in the terminal mouse protocol.
const (
	mouseLeft      = 0
	mouseMiddle    = 1
	mouseRight     = 2
	mouseNoButton  = 3
	mouseWheelUp   = 64
	mouseWheelDown = 65
)

// mouseBytes returns the bytes that report the given mouse event to the
// program, for the mouse mode that it has set, or nil if it is not
// reported. The row and column are 0-based.
func mouseBytes(md *vt.Modes, action mouseAction, button, row, col int, mods key.Modifiers) []byte {
	switch md.Mouse {
	case vt.MouseNone:
		return nil
	case vt.MouseX10:
		if action != mousePress {
			return nil
		}
	case vt.MouseNormal:
		if action == mouseMotion {
			return nil
		}
	case vt.MouseButton:
		if action == mouseMotion && button == mouseNoButton {
			return nil
		}
	}
	b := button
	if md.Mouse != vt.MouseX10 {
		if mods.HasFlag(key.Shift) {
			b += 4
		}
		if mods.HasFlag(key.Alt) {
			b += 8
		}
		if mods.HasFlag(key.Control) {
			b += 16
		}
	}
	if action == mouseMotion {
		b += 32
	}
	if md.MouseSGR {
		final := 'M'
		if action == mouseRelease {
			final = 'm'
		}
		return fmt.Appendf(nil, "\x1b[<%d;%d;%d%c", b, col+1, row+1, final)
	}
	if action == mouseRelease {
		b = b&^3 | mouseNoButton
	}
	enc := func(v int) byte {
		return byte(min(v+32, 255))
	}
	return []byte{0x1b, '[', 'M', enc(b), enc(col + 1), enc(row + 1)}
}
