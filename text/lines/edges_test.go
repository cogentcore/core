// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"testing"
	"time"

	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVirtualTextView tests the view position of a char within virtual
// text, as for the candidate window of an input method.
func TestVirtualTextView(t *testing.T) {
	ls, vid := NewLinesFromBytes("dummy.txt", 80, []byte("hello world\nsecond line\n"))
	other := ls.NewView(80)
	pos := textpos.Pos{Line: 0, Char: 6}
	ls.SetVirtualText("hint", -1, VirtualText{Pos: pos, Text: plain("ab")})
	ls.SetVirtualText("ime", vid, VirtualText{Pos: pos, Text: plain("にほんご")})

	vp, ok := ls.VirtualTextView(vid, "ime", pos, 2)
	require.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 0, Char: 6 + 2 + 2}, vp, "after the hint before it")
	vp, ok = ls.VirtualTextView(vid, "ime", pos, 100)
	require.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 0, Char: 6 + 2 + 4}, vp, "at most its end")
	_, ok = ls.VirtualTextView(other, "ime", pos, 0)
	assert.False(t, ok, "not shown in the other view")
	_, ok = ls.VirtualTextView(vid, "none", pos, 0)
	assert.False(t, ok)
	_, ok = ls.VirtualTextView(vid, "ime", textpos.Pos{Line: 1}, 0)
	assert.False(t, ok)
	_, ok = ls.VirtualTextView(999, "ime", pos, 0)
	assert.False(t, ok, "no such view")
}

// TestVirtualTextClamped tests that virtual text at a char beyond its line
// is put at the end of the line, and at a line beyond the text is dropped.
func TestVirtualTextClamped(t *testing.T) {
	ls, vid := NewLinesFromBytes("dummy.txt", 80, []byte("abc\ndef\n"))
	ls.SetVirtualText("x", vid,
		VirtualText{Pos: textpos.Pos{Line: 0, Char: -5}, Text: plain("<")},
		VirtualText{Pos: textpos.Pos{Line: 1, Char: 50}, Text: plain(">")},
		VirtualText{Pos: textpos.Pos{Line: 9, Char: 0}, Text: plain("!")})
	assert.Equal(t, []string{"<abc", "def>"}, viewText(ls, vid))

	empty, evid := NewLinesFromBytes("dummy.txt", 80, nil)
	empty.SetVirtualText("x", evid, VirtualText{Pos: textpos.Pos{Line: 0, Char: 3}, Text: plain("hi")},
		VirtualText{Pos: textpos.Pos{Line: 1}, Text: plain("no")})
	assert.Len(t, empty.VirtualText("x", evid), 1, "at the start of the empty text")
}

// TestInsertAtEnd tests inserting text at the start of the line after the
// last one, which adds a line, and a rectangle of text beyond the end,
// which adds lines for it.
func TestInsertAtEnd(t *testing.T) {
	ls, vid := NewLinesFromBytes("dummy.txt", 80, []byte("abc"))
	n := ls.NumLines()
	ls.InsertText(textpos.Pos{Line: n}, []rune("new"))
	assert.Equal(t, n+1, ls.NumLines())
	assert.Equal(t, "new", string(ls.Line(n)))
	assert.Equal(t, []string{"abc", "new"}, viewText(ls, vid), "the view has both lines")

	rect := &textpos.Edit{Region: textpos.NewRegion(2, 1, 4, 3), Text: [][]rune{[]rune("xy"), []rune("zw"), []rune("uv")}, Rect: true}
	ls.InsertTextRect(rect)
	assert.Equal(t, 5, ls.NumLines())
	assert.Equal(t, " zw", string(ls.Line(3)), "padded to the start column")
	assert.Equal(t, " uv", string(ls.Line(4)))
	assert.Equal(t, []string{"abc", "new", " xy", " zw", " uv"}, viewText(ls, vid))
}

// TestStartDeferredMarkup tests starting the syntax highlighting markup
// that was deferred until the text is first shown.
func TestStartDeferredMarkup(t *testing.T) {
	ls, _ := NewLinesFromBytes("dummy.go", 80, []byte("package main\n\nfunc main() {}\n"))
	ls.StartDeferredMarkup() // not deferred: nothing to do
	ls.Lock()
	ls.markupDeferred = true
	for ln := range ls.hiTags {
		ls.hiTags[ln] = nil
	}
	ls.Unlock()
	ls.StartDeferredMarkup()
	assert.Eventually(t, func() bool {
		ls.Lock()
		defer ls.Unlock()
		return !ls.markupDeferred && len(ls.hiTags[0]) > 0
	}, 5*time.Second, 10*time.Millisecond, "marked up")
}
