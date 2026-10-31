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
}
