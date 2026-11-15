// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"image"
	"regexp"
	"strings"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestLines returns new lines with the given text, as a go file,
// with a view of the given width.
func newTestLines(t *testing.T, src string) (*Lines, int) {
	t.Helper()
	lns, vid := NewLinesFromBytes("test.go", 80, []byte(src))
	return lns, vid
}

// TestLinesAccess tests the basic accessors of the lines.
func TestLinesAccess(t *testing.T) {
	lns, vid := newTestLines(t, "abc\ndéf\n\nlast")
	assert.Equal(t, 4, lns.NumLines())
	assert.True(t, lns.IsValidLine(3))
	assert.False(t, lns.IsValidLine(4))
	assert.False(t, lns.IsValidLine(-1))
	assert.Equal(t, []rune("déf"), lns.Line(1))
	assert.Nil(t, lns.Line(10))
	assert.Equal(t, 'é', lns.LineChar(1, 1))
	assert.Equal(t, rune(0), lns.LineChar(1, 10), "0 when invalid")
	assert.Equal(t, []string{"abc", "déf", "", "last"}, lns.Strings(false))
	assert.Equal(t, []string{"abc\n", "déf\n", "\n", "last\n"}, lns.Strings(true))
	assert.Equal(t, textpos.Pos{Line: 3, Char: 4}, lns.EndPos())
	assert.True(t, lns.IsValidPos(textpos.Pos{Line: 3, Char: 4}))
	assert.False(t, lns.IsValidPos(textpos.Pos{Line: 3, Char: 5}))
	assert.Equal(t, textpos.Pos{Line: 3, Char: 4}, lns.ValidPos(textpos.Pos{Line: 9, Char: 9}))
	assert.Equal(t, textpos.Pos{Line: 0, Char: 3}, lns.ValidPos(textpos.Pos{Line: 0, Char: 9}))
	assert.Equal(t, textpos.Pos{}, lns.ValidPos(textpos.Pos{Line: -2, Char: -1}))

	assert.Equal(t, 80, lns.Width(vid))
	assert.False(t, lns.SetWidth(vid, 80), "unchanged")
	assert.True(t, lns.SetWidth(vid, 40))
	assert.Equal(t, 40, lns.Width(vid))
	assert.NotNil(t, lns.FontStyle())
	fs := rich.NewStyle()
	lns.SetFontStyle(fs)
	assert.Equal(t, fs, lns.FontStyle())

	lns.SetTextLines([][]byte{[]byte("one"), []byte("two")})
	assert.Equal(t, "one\ntwo\n", lns.String())
	assert.Equal(t, 0, lns.LineLexDepth(0))
	assert.Nil(t, lns.HiTags(9))
}

// TestLinesChanged tests the changed state and its events.
func TestLinesChanged(t *testing.T) {
	lns, vid := newTestLines(t, "abc")
	changes, inputs := 0, 0
	lns.OnChange(vid, func(e events.Event) { changes++ })
	lns.OnInput(vid, func(e events.Event) { inputs++ })
	assert.False(t, lns.IsChanged())
	lns.InsertText(textpos.Pos{}, []rune("x"))
	assert.True(t, lns.IsChanged())
	lns.SetChanged(false)
	assert.False(t, lns.IsChanged())
	lns.SendChange()
	assert.Equal(t, 1, changes)
	lns.SendInput()
	assert.Greater(t, inputs, 0)
}

// TestLinesInsertDelete tests inserting text as bytes and lines, regions,
// and rectangular regions.
func TestLinesInsertDelete(t *testing.T) {
	lns, _ := newTestLines(t, "abcd\nefgh\nijkl")
	lns.InsertTextBytes(textpos.Pos{Line: 1, Char: 2}, []byte("XY\nZ"))
	assert.Equal(t, "abcd\nefXY\nZgh\nijkl\n", lns.String())
	lns.Undo()
	assert.Equal(t, "abcd\nefgh\nijkl\n", lns.String())

	lns.InsertTextLines(textpos.Pos{Line: 0, Char: 4}, [][]rune{[]rune("1"), []rune("2")})
	assert.Equal(t, "abcd1\n2\nefgh\nijkl\n", lns.String())
	lns.Undo()

	reg := lns.Region(textpos.Pos{Line: 0, Char: 1}, textpos.Pos{Line: 1, Char: 2})
	require.NotNil(t, reg)
	assert.Equal(t, "bcd\nef", string(reg.ToBytes()))

	rect := lns.RegionRect(textpos.Pos{Line: 0, Char: 1}, textpos.Pos{Line: 2, Char: 3})
	require.NotNil(t, rect)
	assert.Equal(t, [][]rune{[]rune("bc"), []rune("fg"), []rune("jk")}, rect.Text)

	del := lns.DeleteTextRect(textpos.Pos{Line: 0, Char: 1}, textpos.Pos{Line: 2, Char: 3})
	assert.Equal(t, "ad\neh\nil\n", lns.String())
	lns.InsertTextRect(del)
	assert.Equal(t, "abcd\nefgh\nijkl\n", lns.String())
}

// TestLinesUndoControl tests turning undo off and resetting it.
func TestLinesUndoControl(t *testing.T) {
	lns, _ := newTestLines(t, "abc")
	lns.SetUndoOn(false)
	lns.InsertText(textpos.Pos{}, []rune("x"))
	assert.Empty(t, lns.Undo(), "nothing recorded with undo off")
	assert.Equal(t, "xabc\n", lns.String())
	lns.SetUndoOn(true)
	lns.NewUndoGroup()
	lns.InsertText(textpos.Pos{}, []rune("y"))
	lns.UndoReset()
	assert.Empty(t, lns.Undo(), "reset")
	assert.Equal(t, "yxabc\n", lns.String())
}

// TestLinesMoves tests moving positions by characters, words, and lines.
func TestLinesMoves(t *testing.T) {
	lns, vid := newTestLines(t, "one two\nthree")
	p := textpos.Pos{}
	assert.Equal(t, textpos.Pos{Char: 2}, lns.MoveForward(p, 2))
	assert.Equal(t, textpos.Pos{Line: 1, Char: 0}, lns.MoveForward(textpos.Pos{Char: 7}, 1), "wraps to the next line")
	assert.Equal(t, textpos.Pos{Char: 7}, lns.MoveBackward(textpos.Pos{Line: 1}, 1))
	assert.Equal(t, textpos.Pos{Char: 3}, lns.MoveForwardWord(p, 1))
	assert.Equal(t, textpos.Pos{Char: 3}, lns.MoveBackwardWord(textpos.Pos{Char: 7}, 1), "to the space before the word")
	assert.Equal(t, textpos.Pos{Line: 0, Char: 0}, lns.MoveLineStart(vid, textpos.Pos{Char: 5}))
	assert.Equal(t, textpos.Pos{Line: 0, Char: 7}, lns.MoveLineEnd(vid, textpos.Pos{Char: 2}))

	assert.True(t, lns.TransposeChar(vid, textpos.Pos{Char: 1}))
	assert.Equal(t, "noe two", string(lns.Line(0)))
}

// TestLinesWords tests finding words at and before positions.
func TestLinesWords(t *testing.T) {
	lns, _ := newTestLines(t, "hello world")
	assert.True(t, lns.IsWordEnd(textpos.Pos{Char: 5}))
	assert.False(t, lns.IsWordEnd(textpos.Pos{Char: 2}))
	assert.True(t, lns.IsWordMiddle(textpos.Pos{Char: 2}))
	assert.False(t, lns.IsWordMiddle(textpos.Pos{Char: 5}))
	wa := lns.WordAt(textpos.Pos{Char: 8})
	assert.Equal(t, textpos.Pos{Char: 6}, wa.Start)
	assert.Equal(t, textpos.Pos{Char: 11}, wa.End)
	wb := lns.WordBefore(textpos.Pos{Char: 9})
	require.NotNil(t, wb)
	assert.Equal(t, "wor", string(wb.ToBytes()))
}

// TestLinesPosHistory tests saving positions in the history.
func TestLinesPosHistory(t *testing.T) {
	lns, _ := newTestLines(t, "a\nb\nc")
	assert.True(t, lns.PosHistorySave(textpos.Pos{Line: 1}))
	assert.False(t, lns.PosHistorySave(textpos.Pos{Line: 1}), "the same position is not saved again")
	assert.True(t, lns.PosHistorySave(textpos.Pos{Line: 2}))
	assert.Equal(t, 2, lns.PosHistoryLen())
	p, ok := lns.PosHistoryAt(1)
	assert.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 2}, p)
	_, ok = lns.PosHistoryAt(5)
	assert.False(t, ok)
}

// TestLinesTags tests adding and removing tags, and the lexical queries.
func TestLinesTags(t *testing.T) {
	lns, _ := newTestLines(t, "package x\n\n// note\nvar s = \"str\"\n")
	assert.True(t, lns.InComment(textpos.Pos{Line: 2, Char: 4}))
	assert.False(t, lns.InComment(textpos.Pos{Line: 3, Char: 1}))
	assert.True(t, lns.InLitString(textpos.Pos{Line: 3, Char: 10}))
	assert.False(t, lns.InLitString(textpos.Pos{Line: 3, Char: 1}))
	assert.True(t, lns.InTokenCode(textpos.Pos{Line: 3, Char: 1}))
	lx, _ := lns.HiTagAtPos(textpos.Pos{Line: 3, Char: 1})
	require.NotNil(t, lx)
	assert.True(t, lns.InTokenSubCat(textpos.Pos{Line: 3, Char: 10}, token.LitStr))
	assert.NotEmpty(t, lns.LexObjPathString(3, lx))

	lns.AddTag(0, 0, 7, token.Error)
	reg, ok := lns.RemoveTag(textpos.Pos{Line: 0, Char: 2}, token.Error)
	assert.True(t, ok)
	assert.Equal(t, 7, reg.End)
	_, ok = lns.RemoveTag(textpos.Pos{Line: 0, Char: 2}, token.Error)
	assert.False(t, ok)

	tbe := lns.Region(textpos.Pos{Line: 3, Char: 0}, textpos.Pos{Line: 3, Char: 3})
	lns.AddTagEdit(tbe, token.Error)
	tags := lns.AdjustedTags(0)
	lns.SetTags(0, tags)
	assert.Equal(t, tags, lns.AdjustedTagsLine(tags, 0))
	assert.True(t, lns.MarkupLines(0, 3))
	lns.StartDelayedReMarkup()
	lns.StopDelayedReMarkup()
}

// TestLinesIndentComment tests indenting and commenting lines.
func TestLinesIndentComment(t *testing.T) {
	lns, _ := newTestLines(t, "func f() {\nx := 1\nif x {\ny()\n}\n}\n")
	lns.IndentLine(1, 1)
	assert.Equal(t, "\tx := 1", string(lns.Line(1)))
	lns.IndentLine(1, 0)
	assert.Equal(t, "x := 1", string(lns.Line(1)))

	lns.AutoIndentRegion(1, 6)
	assert.Equal(t, "func f() {\n\tx := 1\n\tif x {\n\t\ty()\n\t}\n}\n", lns.String())
	tbe, lev, ch := lns.AutoIndent(3)
	assert.Nil(t, tbe, "already indented")
	assert.Equal(t, 2, lev)
	assert.Equal(t, 2, ch)

	lns.CommentRegion(1, 3)
	assert.Equal(t, "\t// x := 1", string(lns.Line(1)))
	assert.Equal(t, "\t// if x {", string(lns.Line(2)))
	assert.True(t, lns.InComment(textpos.Pos{Line: 1, Char: 6}))
	lns.CommentRegion(1, 3)
	assert.Equal(t, "\tx := 1", string(lns.Line(1)), "toggles back")

	lns.TabsToSpaces(1, 3)
	assert.Equal(t, "    x := 1", string(lns.Line(1)))
	lns.SpacesToTabs(1, 3)
	assert.Equal(t, "\tx := 1", string(lns.Line(1)))
}

// TestLinesJoinParaLines tests joining paragraphs into one line each.
func TestLinesJoinParaLines(t *testing.T) {
	lns, _ := newTestLines(t, "one\ntwo\n\nthree\nfour")
	lns.JoinParaLines(0, 4)
	assert.Equal(t, "one two\n\nthree four\n", lns.String())
	w, l := lns.CountWordsLinesRegion(textpos.Region{End: lns.EndPos()})
	assert.Equal(t, 4, w)
	assert.Equal(t, 3, l)
}

// TestLinesSearch tests searching and matching braces.
func TestLinesSearch(t *testing.T) {
	lns, _ := newTestLines(t, "Foo foo\n(fOo [x])\nend")
	n, ms := lns.Search([]byte("foo"), true, false)
	assert.Equal(t, 3, n)
	assert.Len(t, ms, 3)
	n, _ = lns.Search([]byte("foo"), false, false)
	assert.Equal(t, 1, n)
	n, _ = lns.SearchRegexp(regexp.MustCompile(`[fF]oo`))
	assert.Equal(t, 2, n)

	p, ok := lns.BraceMatch(textpos.Pos{Line: 1, Char: 0})
	assert.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 1, Char: 8}, p)
	p, ok = lns.BraceMatchRune(']', textpos.Pos{Line: 1, Char: 7})
	assert.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 1, Char: 5}, p)
}

// TestLinesLineColors tests setting and deleting line colors.
func TestLinesLineColors(t *testing.T) {
	lns, _ := newTestLines(t, "a\nb")
	_, ok := lns.LineColor(1)
	assert.False(t, ok)
	var c image.Image = colors.Uniform(colors.Red)
	lns.SetLineColor(1, c)
	got, ok := lns.LineColor(1)
	assert.True(t, ok)
	assert.Equal(t, c, got)
	lns.DeleteLineColor(1)
	_, ok = lns.LineColor(1)
	assert.False(t, ok)
	lns.DeleteLineColor(-1) // deletes all
}

// TestLinesDiffs tests diffs between lines and patching.
func TestLinesDiffs(t *testing.T) {
	a, _ := newTestLines(t, "one\ntwo\nthree")
	b, _ := newTestLines(t, "one\n2\nthree\nfour")
	ds := a.Diffs(b)
	assert.NotEmpty(t, ds)
	assert.NotEmpty(t, ds.String())
	i, op := ds.DiffForLine(1)
	assert.GreaterOrEqual(t, i, 0)
	assert.NotEmpty(t, DiffOpString(op))
	un := string(a.DiffsUnified(b, 1))
	assert.Contains(t, un, "-two")
	assert.Contains(t, un, "+2")
	assert.True(t, a.PatchFrom(b, ds))
	assert.Equal(t, b.String(), a.String())
}

// TestLinesUtil tests the line utility functions.
func TestLinesUtil(t *testing.T) {
	assert.Equal(t, []string{"a\n", "b\n"}, BytesToLineStrings([]byte("a\nb"), true))
	assert.Equal(t, [][]byte{[]byte("a"), []byte("b")}, StringLinesToByteLines([]string{"a", "b"}))
	w, l := CountWordsLines(strings.NewReader("one two\nthree\n"))
	assert.Equal(t, 3, w)
	assert.Equal(t, 2, l)

	comLn, comSt, comEd := KnownComments("x.go")
	assert.Equal(t, "// ", comLn)
	assert.Equal(t, "/* ", comSt)
	assert.Equal(t, " */", comEd)

	src := [][]byte{[]byte("x := 1"), []byte(""), []byte("// doc"), []byte("// more"), []byte("func f() {}")}
	assert.Equal(t, 2, PreCommentStart(src, 4, "//", "/*", "*/", 5))
	assert.Equal(t, 3, PreCommentStart(src, 4, "//", "/*", "*/", 0), "only looks back lnBack lines")
	src = [][]byte{[]byte("/* a"), []byte("b */"), []byte("func f() {}")}
	assert.Equal(t, 0, PreCommentStart(src, 2, "//", "/*", "*/", 5))

	var s Settings
	s.Defaults()
	s.CommentLine = "#"
	st, ed := s.CommentStrings()
	assert.Equal(t, "#", st)
	assert.Equal(t, "", ed)
	s.CommentLine = ""
	s.CommentStart, s.CommentEnd = "<!--", "-->"
	st, ed = s.CommentStrings()
	assert.Equal(t, "<!--", st)
	assert.Equal(t, "-->", ed)
	assert.False(t, s.ConfigKnown(fileinfo.Unknown))
	assert.True(t, s.ConfigKnown(fileinfo.Go))
}

// TestLinesBraceMatchEdges tests matching braces whose match is on the
// last or first line.
func TestLinesBraceMatchEdges(t *testing.T) {
	lns, _ := newTestLines(t, "(\na\n)")
	p, ok := lns.BraceMatch(textpos.Pos{Line: 0})
	assert.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 2}, p)
	p, ok = lns.BraceMatch(textpos.Pos{Line: 2})
	assert.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 0}, p)
}

// TestLinesAutoIndentFirst tests that the first line is not indented, as
// it has no previous line.
func TestLinesAutoIndentFirst(t *testing.T) {
	lns, _ := newTestLines(t, "func f() {\nx := 1\n}\n")
	lns.AutoIndent(0)
	assert.Equal(t, "func f() {", string(lns.Line(0)))
}

// TestPreCommentStartFileStart tests finding comment lines that start
// at the first line.
func TestPreCommentStartFileStart(t *testing.T) {
	src := [][]byte{[]byte("// doc"), []byte("// more"), []byte("func f() {}")}
	assert.Equal(t, 0, PreCommentStart(src, 2, "//", "/*", "*/", 5))
}

// TestLinesTransposeWord tests swapping words.
func TestLinesTransposeWord(t *testing.T) {
	tests := []struct {
		src  string
		at   int
		want string
		end  int
	}{
		{"one two three", 5, "one three two", 13},      // in a word
		{"one two three", 4, "one three two", 13},      // at the start of a word
		{"one two three", 3, "two one three", 7},       // just after a word
		{"one two three", 13, "one three two", 13},     // at the end: the last two words
		{"  one, two", 0, "  two, one", 10},            // before the first word
		{"a(b) c", 2, "a(c) b", 6},                     // punctuation stays
		{"x := y.Z", 0, "y := x.Z", 6},                 // non-word runes between
		{"one   two\nthree", 1, "two   one\nthree", 9}, // within the line
	}
	for _, tt := range tests {
		lns, _ := newTestLines(t, tt.src)
		pos, ok := lns.TransposeWord(textpos.Pos{Char: tt.at})
		assert.True(t, ok, tt.src)
		assert.Equal(t, tt.want, lns.String()[:len(tt.want)], tt.src)
		assert.Equal(t, textpos.Pos{Char: tt.end}, pos, tt.src)
	}
	for _, src := range []string{"one", "", "  ,  "} {
		lns, _ := newTestLines(t, src)
		_, ok := lns.TransposeWord(textpos.Pos{})
		assert.False(t, ok, src)
	}
	lns, _ := newTestLines(t, "a b")
	_, ok := lns.TransposeWord(textpos.Pos{Line: 5})
	assert.False(t, ok, "an invalid position")
}
