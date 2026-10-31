// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

// imeLayer is the virtual text layer of the composition of an input method.
const imeLayer = "ime"

// composition is the state of text being composed with an input method (IME),
// which is shown as virtual text at the cursor until it is committed.
type composition struct {

	// active is whether text is being composed.
	active bool

	// pos is the source position of the composition.
	pos textpos.Pos

	// start and end are the range of the selected part of the
	// composition, in runes, where end is the input method cursor.
	start, end int
}

// compose handles a [events.Composition] event from an input method,
// showing the text being composed at the cursor as virtual text, or
// ending the composition if it is empty, in which case the committed
// text follows as KeyChord events.
func (ed *Editor) compose(e *events.TextComposition) {
	if ed.Lines == nil || ed.IsReadOnly() {
		return
	}
	if e.Text == "" {
		ed.endComposition()
		return
	}
	cp := &ed.composition
	if !cp.active {
		if ed.HasSelection() { // typing replaces the selection
			tbe := ed.deleteSelection()
			ed.CursorPos = tbe.AdjustPos(ed.CursorPos, textpos.AdjustPosDelStart)
		}
		cp.active = true
		cp.pos = ed.CursorPos
	}
	txt := []rune(e.Text)
	cp.start = max(0, min(e.Start, len(txt)))
	cp.end = max(cp.start, min(e.End, len(txt)))
	ed.SetVirtualText(imeLayer, lines.VirtualText{Pos: cp.pos, Text: ed.compositionText(txt, cp.start, cp.end)})
	ed.NeedsRender()
}

// endComposition ends any composition, removing its text.
func (ed *Base) endComposition() {
	if !ed.composition.active {
		return
	}
	ed.composition.active = false
	ed.ClearVirtualText(imeLayer)
	ed.NeedsRender()
}

// compositionText returns the text being composed as rich text, which is
// underlined, with the selected part, such as the clause being converted,
// also highlighted.
func (ed *Base) compositionText(txt []rune, start, end int) rich.Text {
	sty := ed.CompositionStyle()
	var tx rich.Text
	tx.AddSpan(sty, txt[:start])
	if end > start {
		sel := *sty
		sel.SetBackground(colors.ToUniform(colors.Scheme.Select.Container))
		tx.AddSpan(&sel, txt[start:end])
	}
	tx.AddSpan(sty, txt[end:])
	return tx
}

// updateTextInput updates the text input state of the window for the input
// method, with the position of the cursor, or that of the input method
// within the text being composed, where its candidate window is shown.
func (ed *Base) updateTextInput() {
	if ed.IsReadOnly() || !ed.StateIs(states.Focused) {
		return
	}
	pos := ed.charStartPos(ed.CursorPos)
	if cp := &ed.composition; cp.active {
		if p, ok := ed.VirtualCharStartPos(imeLayer, cp.pos, cp.end); ok {
			pos = p
		}
	}
	min := pos.ToPointFloor()
	ed.SetTextInput(image.Rectangle{Min: min, Max: min.Add(ed.charSize.ToPointCeil())})
}
