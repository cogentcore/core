// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import "image"

// TextInput is the state of text input in a window, which the input
// method (IME) of the operating system uses, such as for composing
// Chinese, Japanese and Korean text (see [Window.SetTextInput]).
type TextInput struct {

	// Caret is the rectangle of the text cursor, or of the cursor of
	// the input method within the composition, in window coordinates
	// (the same as those of events). The input method shows its
	// candidate window next to it, typically below it.
	Caret image.Rectangle

	// Before is the text before the cursor, on its line, up to
	// [TextInputBeforeMax] runes, or before the text being composed, which
	// a virtual keyboard uses for its predictions and corrections (on iOS).
	// It is empty for the text of a password.
	Before string
}

// TextInputBeforeMax is the most runes of the text before the cursor in
// [TextInput.Before].
const TextInputBeforeMax = 200

// TextInputBefore returns the [TextInput.Before] text of the given text of
// the line of the cursor, up to the cursor: at most [TextInputBeforeMax]
// runes, after any newline.
func TextInputBefore(line []rune) string {
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == '\n' {
			line = line[i+1:]
			break
		}
	}
	if n := len(line); n > TextInputBeforeMax {
		line = line[n-TextInputBeforeMax:]
	}
	return string(line)
}
