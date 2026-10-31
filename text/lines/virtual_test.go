// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// viewText returns the text of each view line of the given view.
func viewText(ls *Lines, vid int) []string {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	txt := make([]string, vw.viewLines)
	for vl := range vw.viewLines {
		txt[vl] = string(vw.markup[vl].Join())
	}
	return txt
}

// checkRoundTrip checks that every source position maps to
// the view and back to the same position.
func checkRoundTrip(t *testing.T, ls *Lines, vid int) {
	t.Helper()
	for ln := range ls.NumLines() {
		for ch := range ls.LineLen(ln) + 1 {
			pos := textpos.Pos{Line: ln, Char: ch}
			vp := ls.PosToView(vid, pos)
			assert.Equal(t, pos, ls.PosFromView(vid, vp), "view pos %v", vp)
		}
	}
}

// eventCounts counts the events sent to the given view.
type eventCounts struct {
	change, input, virtual int
}

func countEvents(ls *Lines, vid int) *eventCounts {
	ec := &eventCounts{}
	ls.OnChange(vid, func(e events.Event) { ec.change++ })
	ls.OnInput(vid, func(e events.Event) { ec.input++ })
	ls.OnVirtualText(vid, func(e events.Event) { ec.virtual++ })
	return ec
}

// TestVirtualIME tests virtual text for the composition of an input method:
// it is shown only in its view, at the cursor, and is then committed as
// a single edit.
func TestVirtualIME(t *testing.T) {
	ls, vid := NewLinesFromBytes("dummy.txt", 80, []byte("hello world\nsecond line\n"))
	other := ls.NewView(80)
	ec := countEvents(ls, vid)

	ls.SetVirtualText("ime", vid, VirtualText{Pos: textpos.Pos{Line: 0, Char: 6}, Text: []rune("にほんご")})
	assert.Equal(t, []string{"hello にほんごworld", "second line"}, viewText(ls, vid))
	assert.Equal(t, []string{"hello world", "second line"}, viewText(ls, other))
	assert.Equal(t, "hello world\nsecond line\n", ls.String())
	assert.False(t, ls.IsChanged())
	assert.Equal(t, eventCounts{virtual: 1}, *ec)

	// the cursor is shown before the composition, and
	// view positions within it are at its position
	assert.Equal(t, textpos.Pos{Line: 0, Char: 6}, ls.PosToView(vid, textpos.Pos{Line: 0, Char: 6}))
	assert.Equal(t, textpos.Pos{Line: 0, Char: 11}, ls.PosToView(vid, textpos.Pos{Line: 0, Char: 7}))
	assert.Equal(t, textpos.Pos{Line: 0, Char: 6}, ls.PosFromView(vid, textpos.Pos{Line: 0, Char: 8}))
	assert.Equal(t, textpos.Pos{Line: 0, Char: 7}, ls.PosFromView(vid, textpos.Pos{Line: 0, Char: 11}))
	assert.Equal(t, 16, ls.ViewLineLen(vid, 0))
	checkRoundTrip(t, ls, vid)

	// the composition is converted
	ls.SetVirtualText("ime", vid, VirtualText{Pos: textpos.Pos{Line: 0, Char: 6}, Text: []rune("日本語")})
	assert.Equal(t, []string{"hello 日本語world", "second line"}, viewText(ls, vid))

	// and committed
	ls.ClearVirtualText("ime", vid)
	ls.InsertText(textpos.Pos{Line: 0, Char: 6}, []rune("日本語"))
	assert.Equal(t, []string{"hello 日本語world", "second line"}, viewText(ls, vid))
	assert.Equal(t, "hello 日本語world\nsecond line\n", ls.String())
	assert.Nil(t, ls.VirtualText("ime", vid))
	ls.Undo()
	assert.Equal(t, "hello world\nsecond line\n", ls.String())
}

// TestVirtualInlay tests virtual text for language server inlay hints:
// it is shown in all views, and stays attached to the text after it
// as the text is edited.
func TestVirtualInlay(t *testing.T) {
	ls, vid := NewLinesFromBytes("dummy.txt", 80, []byte("foo(1, 2)\nbar(3)\n"))
	other := ls.NewView(80)
	hint := func(ln, ch int, txt string) VirtualText {
		return VirtualText{Pos: textpos.Pos{Line: ln, Char: ch}, Text: []rune(txt)}
	}
	ls.SetVirtualText("inlay", -1, hint(0, 4, "a: "), hint(0, 7, "b: "), hint(1, 4, "c: "))
	want := []string{"foo(a: 1, b: 2)", "bar(c: 3)"}
	assert.Equal(t, want, viewText(ls, vid))
	assert.Equal(t, want, viewText(ls, other))
	checkRoundTrip(t, ls, vid)

	// text typed at a hint goes before it
	ls.InsertText(textpos.Pos{Line: 0, Char: 4}, []rune("x"))
	assert.Equal(t, []string{"foo(xa: 1, b: 2)", "bar(c: 3)"}, viewText(ls, vid))
	assert.Equal(t, []VirtualText{hint(0, 5, "a: "), hint(0, 8, "b: "), hint(1, 4, "c: ")}, ls.VirtualText("inlay", -1))
	checkRoundTrip(t, ls, vid)

	// a new line moves the hints after it to the new line
	ls.InsertText(textpos.Pos{Line: 0, Char: 7}, []rune("\n"))
	assert.Equal(t, []string{"foo(xa: 1,", " b: 2)", "bar(c: 3)"}, viewText(ls, vid))
	assert.Equal(t, []VirtualText{hint(0, 5, "a: "), hint(1, 1, "b: "), hint(2, 4, "c: ")}, ls.VirtualText("inlay", -1))

	// joining the lines moves them back
	ls.DeleteText(textpos.Pos{Line: 0, Char: 7}, textpos.Pos{Line: 1, Char: 0})
	assert.Equal(t, []string{"foo(xa: 1, b: 2)", "bar(c: 3)"}, viewText(ls, vid))

	// deleting the text around a hint removes it
	ls.DeleteText(textpos.Pos{Line: 0, Char: 4}, textpos.Pos{Line: 0, Char: 6})
	assert.Equal(t, []string{"foo(, b: 2)", "bar(c: 3)"}, viewText(ls, vid))
	assert.Equal(t, []VirtualText{hint(0, 6, "b: "), hint(1, 4, "c: ")}, ls.VirtualText("inlay", -1))

	// deleting lines removes their hints
	ls.DeleteText(textpos.Pos{Line: 0, Char: 8}, textpos.Pos{Line: 1, Char: 6})
	assert.Equal(t, []string{"foo(, b: 2)"}, viewText(ls, vid))
	assert.Equal(t, "foo(, 2)\n", ls.String())
	checkRoundTrip(t, ls, vid)
}

// TestVirtualWrap tests wrapping of lines with virtual text.
func TestVirtualWrap(t *testing.T) {
	ls, vid := NewLinesFromBytes("dummy.txt", 12, []byte("alpha beta gamma delta\n"))
	ls.SetVirtualText("inlay", -1, VirtualText{Pos: textpos.Pos{Line: 0, Char: 6}, Text: []rune("hint: ")})
	assert.Equal(t, []string{"alpha hint: ", "beta gamma ", "delta"}, viewText(ls, vid))
	// the cursor at the hint is shown before it, and the text after it is wrapped
	assert.Equal(t, textpos.Pos{Line: 0, Char: 6}, ls.PosToView(vid, textpos.Pos{Line: 0, Char: 6}))
	assert.Equal(t, textpos.Pos{Line: 1, Char: 1}, ls.PosToView(vid, textpos.Pos{Line: 0, Char: 7}))
	checkRoundTrip(t, ls, vid)
}

// TestVirtualSuggestion tests virtual text for a multi-line suggested
// completion, whose lines are separate view lines that the cursor skips.
func TestVirtualSuggestion(t *testing.T) {
	ls, vid := NewLinesFromBytes("dummy.txt", 80, []byte("func f() {\n}\n"))
	sug := VirtualText{Pos: textpos.Pos{Line: 0, Char: 10}, Text: []rune("\n  a := 1\n  return a")}
	ls.SetVirtualText("suggestion", vid, sug)
	assert.Equal(t, []string{"func f() {", "  a := 1", "  return a", "}"}, viewText(ls, vid))
	assert.Equal(t, textpos.Pos{Line: 0, Char: 10}, ls.PosFromView(vid, textpos.Pos{Line: 1, Char: 3}))
	// the cursor at the end of the line is shown before the suggestion
	assert.Equal(t, textpos.Pos{Line: 0, Char: 10}, ls.PosToView(vid, textpos.Pos{Line: 0, Char: 10}))
	checkRoundTrip(t, ls, vid)

	// the cursor skips the suggestion lines
	assert.Equal(t, textpos.Pos{Line: 1, Char: 1}, ls.MoveDown(vid, textpos.Pos{Line: 0, Char: 3}, 1, 3))
	assert.Equal(t, textpos.Pos{Line: 0, Char: 0}, ls.MoveUp(vid, textpos.Pos{Line: 1, Char: 0}, 1, 0))

	// accepting it inserts exactly its text
	ls.ClearVirtualText("suggestion", vid)
	assert.Equal(t, []string{"func f() {", "}"}, viewText(ls, vid))
	ls.InsertText(sug.Pos, sug.Text)
	assert.Equal(t, "func f() {\n  a := 1\n  return a\n}\n", ls.String())

	// in the middle of a line, the rest of the line follows its last line
	ls.SetVirtualText("suggestion", vid, VirtualText{Pos: textpos.Pos{Line: 0, Char: 5}, Text: []rune("X\nY")})
	assert.Equal(t, []string{"func X", "Yf() {", "  a := 1", "  return a", "}"}, viewText(ls, vid))
	assert.Equal(t, textpos.Pos{Line: 1, Char: 2}, ls.PosToView(vid, textpos.Pos{Line: 0, Char: 6}))
	checkRoundTrip(t, ls, vid)
}
