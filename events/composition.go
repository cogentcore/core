// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import "fmt"

// TextComposition is a [Composition] event, which is sent to the focused
// widget while the user composes text with an input method (IME), such
// as for Chinese, Japanese and Korean text, or for accented characters.
// The widget shows the text at its cursor, typically underlined, without
// adding it to its text, and reports the position of the cursor for the
// candidate window of the input method (see [system.Window.SetTextInput]).
// The composition ends with an event with empty text, after which any
// committed text is sent as KeyChord events.
type TextComposition struct {
	Base

	// Text is the text being composed, which is not yet committed.
	// It is empty when the composition ends.
	Text string

	// Start and End are the range of the selected part of Text,
	// in runes, such as the clause being converted, which is the
	// position of the cursor of the input method if they are equal.
	Start, End int
}

// NewTextComposition returns a new [Composition] event with
// the given text, and range of its selected part, in runes.
func NewTextComposition(text string, start, end int) *TextComposition {
	ev := &TextComposition{Text: text, Start: start, End: end}
	ev.Typ = Composition
	return ev
}

func (ev *TextComposition) HasPos() bool {
	return false
}

func (ev *TextComposition) NeedsFocus() bool {
	return true
}

func (ev *TextComposition) String() string {
	return fmt.Sprintf("%v{Text: %q, Start: %d, End: %d, Time: %v}", ev.Type(), ev.Text, ev.Start, ev.End, ev.Time().Format("04:05"))
}

// Composition sends a [Composition] event with the given text, and range
// of its selected part, in runes. An empty text ends the composition.
func (es *Source) Composition(text string, start, end int) {
	ev := NewTextComposition(text, start, end)
	ev.Init()
	es.Deque.Send(ev)
}
