// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"unicode"

	"cogentcore.org/core/events/key"
	"cogentcore.org/core/text/terminal/vt"
)

// keyBytes returns the bytes to send to the terminal program for the
// given key chord, or nil if the key does not send anything. If altMeta
// is set, Alt (Option) with a character sends Escape and then the
// character of the key without Alt, which is given by the key code (as
// the rune is the composed character on macOS): for character events,
// which have no code, the code of the last key down event is given.
func keyBytes(rn rune, code key.Codes, mods key.Modifiers, md *vt.Modes, altMeta bool) []byte {
	shift := mods.HasFlag(key.Shift)
	ctrl := mods.HasFlag(key.Control)
	alt := mods.HasFlag(key.Alt)
	mod := 1
	if shift {
		mod++
	}
	if alt {
		mod += 2
	}
	if ctrl {
		mod += 4
	}
	esc := func(s string) []byte {
		if alt && altMeta {
			return []byte("\x1b" + s)
		}
		return []byte(s)
	}
	// cursor returns the code for a cursor key, which has the SS3
	// form in application mode.
	cursor := func(final byte) []byte {
		if mod > 1 {
			return fmt.Appendf(nil, "\x1b[1;%d%c", mod, final)
		}
		if md.AppCursorKeys {
			return []byte{0x1b, 'O', final}
		}
		return []byte{0x1b, '[', final}
	}
	// tilde returns the code for a key with the CSI n ~ form.
	tilde := func(n int) []byte {
		if mod > 1 {
			return fmt.Appendf(nil, "\x1b[%d;%d~", n, mod)
		}
		return fmt.Appendf(nil, "\x1b[%d~", n)
	}
	// function returns the code for F1-F4.
	function := func(final byte) []byte {
		if mod > 1 {
			return fmt.Appendf(nil, "\x1b[1;%d%c", mod, final)
		}
		return []byte{0x1b, 'O', final}
	}

	switch code {
	case key.CodeReturnEnter:
		return esc("\r")
	case key.CodeKeypadEnter:
		if md.AppKeypad {
			return []byte("\x1bOM")
		}
		return esc("\r")
	case key.CodeEscape:
		return esc("\x1b")
	case key.CodeBackspace:
		if ctrl {
			return esc("\x08")
		}
		return esc("\x7f")
	case key.CodeTab:
		if shift {
			return []byte("\x1b[Z")
		}
		return esc("\t")
	case key.CodeUpArrow:
		return cursor('A')
	case key.CodeDownArrow:
		return cursor('B')
	case key.CodeRightArrow:
		return cursor('C')
	case key.CodeLeftArrow:
		return cursor('D')
	case key.CodeHome:
		return cursor('H')
	case key.CodeEnd:
		return cursor('F')
	case key.CodeInsert:
		return tilde(2)
	case key.CodeDelete:
		return tilde(3)
	case key.CodePageUp:
		return tilde(5)
	case key.CodePageDown:
		return tilde(6)
	case key.CodeF1:
		return function('P')
	case key.CodeF2:
		return function('Q')
	case key.CodeF3:
		return function('R')
	case key.CodeF4:
		return function('S')
	case key.CodeF5:
		return tilde(15)
	case key.CodeF6:
		return tilde(17)
	case key.CodeF7:
		return tilde(18)
	case key.CodeF8:
		return tilde(19)
	case key.CodeF9:
		return tilde(20)
	case key.CodeF10:
		return tilde(21)
	case key.CodeF11:
		return tilde(23)
	case key.CodeF12:
		return tilde(24)
	}
	if rn == 0 {
		return nil
	}
	if alt && altMeta {
		if r, ok := key.CodeRuneMap[code]; ok {
			rn = r
		}
		if !shift {
			rn = unicode.ToLower(rn)
		}
	}
	if ctrl {
		if c, ok := controlChar(rn); ok {
			return esc(string(c))
		}
	}
	return esc(string(rn))
}

// controlChar returns the control character for the given character
// typed with the Control key.
func controlChar(rn rune) (rune, bool) {
	switch {
	case rn >= 'a' && rn <= 'z':
		return rn - 'a' + 1, true
	case rn >= '@' && rn <= '_': // A-Z [ \ ] ^ _
		return rn - '@', true
	case rn == ' ' || rn == '2':
		return 0, true
	case rn >= '3' && rn <= '7':
		return rn - '3' + 0x1b, true
	case rn == '8' || rn == '?':
		return 0x7f, true
	case rn == '/':
		return 0x1f, true
	}
	return 0, false
}
