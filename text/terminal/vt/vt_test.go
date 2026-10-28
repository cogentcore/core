// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// screenLines returns the text of the lines of the screen.
func screenLines(em *Emulator) []string {
	s := make([]string, em.Rows)
	for i, ln := range em.Lines() {
		s[i] = ln.String()
	}
	return s
}

// lineStrings returns the text of the given lines.
func lineStrings(lns []*Line) []string {
	s := make([]string, len(lns))
	for i, ln := range lns {
		s[i] = ln.String()
	}
	return s
}

func cursor(em *Emulator) [2]int {
	return [2]int{em.Cursor.Row, em.Cursor.Col}
}

func TestPrintWrapScroll(t *testing.T) {
	em := NewEmulator(5, 3)
	em.WriteString("abc\r\n12345678")
	assert.Equal(t, []string{"abc", "12345", "678"}, screenLines(em))
	assert.True(t, em.Line(1).Wrapped)
	assert.False(t, em.Line(0).Wrapped)
	assert.Equal(t, [2]int{2, 3}, cursor(em))

	em = NewEmulator(5, 3)
	em.WriteString("abc\r\n12345678\r\nx")
	sc, cleared := em.TakeScrolled()
	assert.False(t, cleared)
	assert.Equal(t, []string{"abc"}, lineStrings(sc))
	assert.Equal(t, []string{"12345", "678", "x"}, screenLines(em))
	assert.Equal(t, 1, em.NumScrolled())

	// pending wrap: a character in the last column does not wrap
	// until the next one
	em = NewEmulator(5, 3)
	em.WriteString("12345")
	assert.Equal(t, [2]int{0, 4}, cursor(em))
	assert.False(t, em.Line(0).Wrapped)
	em.WriteString("\r\n")
	assert.Equal(t, [2]int{1, 0}, cursor(em))
	assert.False(t, em.Line(0).Wrapped, "no wrap with an explicit new line")

	// no auto wrap
	em = NewEmulator(5, 3)
	em.WriteString("\x1b[?7l1234567")
	assert.Equal(t, []string{"12347", "", ""}, screenLines(em))
}

func TestControls(t *testing.T) {
	em := NewEmulator(20, 3)
	em.WriteString("abc\bX\tY\r\tZ")
	assert.Equal(t, "abX     Z", em.Line(0).String(), "Z overwrites Y")
	assert.Equal(t, [2]int{0, 9}, cursor(em))
	em.WriteString("\x1b[3g\r\tq") // clear all tabs
	assert.Equal(t, [2]int{0, 19}, cursor(em))
	em.WriteString("\x1bc")
	assert.Equal(t, []string{"", "", ""}, screenLines(em))
}

func TestCursorErase(t *testing.T) {
	em := NewEmulator(6, 4)
	fill := func() {
		em.WriteString("\x1b[H")
		for i := range 4 {
			em.WriteString(strings.Repeat(string(rune('a'+i)), 6))
			if i < 3 {
				em.WriteString("\r\n")
			}
		}
	}
	fill()
	assert.Equal(t, []string{"aaaaaa", "bbbbbb", "cccccc", "dddddd"}, screenLines(em))
	em.WriteString("\x1b[2;3H\x1b[K")
	assert.Equal(t, []string{"aaaaaa", "bb", "cccccc", "dddddd"}, screenLines(em))
	em.WriteString("\x1b[1K")
	assert.Equal(t, "      ", string(em.Line(1).Runes(6)))
	fill()
	em.WriteString("\x1b[3;4H\x1b[J")
	assert.Equal(t, []string{"aaaaaa", "bbbbbb", "ccc", ""}, screenLines(em))
	fill()
	em.WriteString("\x1b[2;2H\x1b[1J")
	assert.Equal(t, []string{"", "  bbbb", "cccccc", "dddddd"}, screenLines(em))
	fill()
	em.WriteString("\x1b[2J")
	assert.Equal(t, []string{"", "", "", ""}, screenLines(em))
	sc, _ := em.TakeScrolled()
	assert.Equal(t, 4, len(sc), "the text is kept in the scrollback")
	em.WriteString("\x1b[3J")
	_, cleared := em.TakeScrolled()
	assert.True(t, cleared)

	em.WriteString("\x1b[10;10H")
	assert.Equal(t, [2]int{3, 5}, cursor(em), "clamped")
	em.WriteString("\x1b[2A\x1b[3D")
	assert.Equal(t, [2]int{1, 2}, cursor(em))
	em.WriteString("\x1b[5B\x1b[G")
	assert.Equal(t, [2]int{3, 0}, cursor(em))
	em.WriteString("\x1b[2;4f\x1b[d")
	assert.Equal(t, [2]int{0, 3}, cursor(em))
	em.WriteString("\x1b7\x1b[4;4H\x1b8")
	assert.Equal(t, [2]int{0, 3}, cursor(em), "save and restore")
}

func TestInsertDelete(t *testing.T) {
	em := NewEmulator(6, 4)
	em.WriteString("abcdef\x1b[1;3H\x1b[2@")
	assert.Equal(t, "ab  cd", em.Line(0).String())
	em.WriteString("\x1b[3P")
	assert.Equal(t, "ab", em.Line(0).String()[:2])
	assert.Equal(t, "abd", em.Line(0).String())
	em.WriteString("\x1b[4hXY\x1b[4l")
	assert.Equal(t, "abXYd", em.Line(0).String())
	em.WriteString("\x1b[1;2H\x1b[2X")
	assert.Equal(t, "a  Yd", em.Line(0).String())

	em = NewEmulator(3, 4)
	em.WriteString("a\r\nb\r\nc\r\nd\x1b[2;1H\x1b[L")
	assert.Equal(t, []string{"a", "", "b", "c"}, screenLines(em))
	em.WriteString("\x1b[2M")
	assert.Equal(t, []string{"a", "c", "", ""}, screenLines(em))
	sc, _ := em.TakeScrolled()
	assert.Equal(t, 0, len(sc), "deleted lines are not scrollback")
}

func TestScrollRegion(t *testing.T) {
	em := NewEmulator(3, 5)
	em.WriteString("a\r\nb\r\nc\r\nd\r\ne")
	em.WriteString("\x1b[2;4r") // rows 2-4
	assert.Equal(t, [2]int{0, 0}, cursor(em))
	em.WriteString("\x1b[4;1H\nx")
	assert.Equal(t, []string{"a", "c", "d", "x", "e"}, screenLines(em))
	sc, _ := em.TakeScrolled()
	assert.Equal(t, 0, len(sc), "scrolling in a region below the top is not scrollback")
	em.WriteString("\x1b[2;1H\x1bMy")
	assert.Equal(t, []string{"a", "y", "c", "d", "e"}, screenLines(em))
	em.WriteString("\x1b[S")
	assert.Equal(t, []string{"a", "c", "d", "", "e"}, screenLines(em))
	em.WriteString("\x1b[T")
	assert.Equal(t, []string{"a", "", "c", "d", "e"}, screenLines(em))
	em.WriteString("\x1b[r\x1b[5;1H\n")
	sc, _ = em.TakeScrolled()
	assert.Equal(t, []string{"a"}, lineStrings(sc))
}

func TestSGR(t *testing.T) {
	em := NewEmulator(20, 2)
	em.WriteString("\x1b[1;31ma\x1b[22;4;44mb\x1b[0mc\x1b[38;5;200;48;2;1;2;3md\x1b[38:2::10:20:30me\x1b[38:5:9;7mf\x1b[97;100mg\x1b[39;49;27mh")
	cs := em.Line(0).Cells
	assert.Equal(t, Style{Fg: IndexedColor(1), Attrs: Bold}, cs[0].Style)
	assert.Equal(t, Style{Fg: IndexedColor(1), Bg: IndexedColor(4), Attrs: Underline}, cs[1].Style)
	assert.Equal(t, Style{}, cs[2].Style)
	assert.Equal(t, Style{Fg: IndexedColor(200), Bg: RGBColor(1, 2, 3)}, cs[3].Style)
	assert.Equal(t, RGBColor(10, 20, 30), cs[4].Style.Fg)
	assert.Equal(t, Style{Fg: IndexedColor(9), Bg: RGBColor(1, 2, 3), Attrs: Reverse}, cs[5].Style)
	assert.Equal(t, Style{Fg: IndexedColor(15), Bg: IndexedColor(8), Attrs: Reverse}, cs[6].Style)
	assert.Equal(t, Style{}, cs[7].Style)

	em.WriteString("\x1b[44m\x1b[2;1H\x1b[K")
	assert.Equal(t, IndexedColor(4), em.Line(1).Cells[5].Style.Bg, "erase with the background color")

	r, g, b, ok := RGBColor(1, 2, 3).RGB()
	assert.True(t, ok)
	assert.Equal(t, []uint8{1, 2, 3}, []uint8{r, g, b})
	_, ok = RGBColor(1, 2, 3).Index()
	assert.False(t, ok)
}

func TestAltScreen(t *testing.T) {
	em := NewEmulator(5, 3)
	em.WriteString("hello\r\n$ ")
	em.WriteString("\x1b[?1049h")
	assert.True(t, em.IsAltScreen())
	assert.Equal(t, []string{"", "", ""}, screenLines(em))
	em.WriteString("\x1b[Hvi\r\n\r\n\r\n\r\nend")
	sc, _ := em.TakeScrolled()
	assert.Equal(t, 0, len(sc), "no scrollback in the alternate screen")
	em.WriteString("\x1b[?1049l")
	assert.False(t, em.IsAltScreen())
	assert.Equal(t, []string{"hello", "$", ""}, screenLines(em))
	assert.Equal(t, [2]int{1, 2}, cursor(em))
}

func TestWide(t *testing.T) {
	em := NewEmulator(5, 3)
	em.WriteString("a世界")
	cs := em.Line(0).Cells
	assert.Equal(t, uint8(2), cs[1].Width)
	assert.Equal(t, uint8(0), cs[2].Width)
	assert.Equal(t, "a世界", em.Line(0).String())
	assert.Equal(t, [2]int{0, 4}, cursor(em), "pending wrap")
	em.WriteString("b")
	assert.Equal(t, "b", em.Line(1).String())
	em.WriteString("\x1b[1;3Hx") // overwrite the second half of 世
	assert.Equal(t, "a x界", em.Line(0).String())

	em = NewEmulator(4, 3)
	em.WriteString("abc世") // no room at the end of the line
	assert.Equal(t, []string{"abc", "世", ""}, screenLines(em))
	assert.True(t, em.Line(0).Wrapped)

	em = NewEmulator(4, 3)
	em.WriteString("é") // combining accent
	assert.Equal(t, "é", em.Line(0).String())
}

func TestDECGraphics(t *testing.T) {
	em := NewEmulator(10, 2)
	em.WriteString("\x1b(0lqk\x1b(Bx\x1b)0\x0eq\x0fq")
	assert.Equal(t, "┌─┐x─q", em.Line(0).String())
}

func TestReplies(t *testing.T) {
	em := NewEmulator(10, 5)
	var replies []string
	em.Reply = func(b []byte) { replies = append(replies, string(b)) }
	em.WriteString("\x1b[3;4H\x1b[6n\x1b[c\x1b[5n\x1b]11;?\x07\x1b[18t")
	assert.Equal(t, []string{"\x1b[3;4R", "\x1b[?62;22c", "\x1b[0n", "\x1b]11;rgb:ffff/ffff/ffff\x1b\\", "\x1b[8;5;10t"}, replies)
}

func TestModes(t *testing.T) {
	em := NewEmulator(10, 5)
	em.WriteString("\x1b[?1h\x1b[?25l\x1b[?2004h\x1b[?1002h\x1b[?1006h\x1b=\x1b[5 q")
	md := em.Modes
	assert.True(t, md.AppCursorKeys)
	assert.True(t, md.CursorHidden)
	assert.True(t, md.BracketedPaste)
	assert.Equal(t, MouseButton, md.Mouse)
	assert.True(t, md.MouseSGR)
	assert.True(t, md.AppKeypad)
	assert.Equal(t, 5, md.CursorShape)
	em.WriteString("\x1b[?1l\x1b[?25h\x1b[?1002l\x1b>")
	md = em.Modes
	assert.False(t, md.AppCursorKeys)
	assert.False(t, md.CursorHidden)
	assert.Equal(t, MouseNone, md.Mouse)
	assert.False(t, md.AppKeypad)
}

func TestOSC(t *testing.T) {
	em := NewEmulator(20, 5)
	nchanged := 0
	em.Changed = func() { nchanged++ }
	em.WriteString("\x1b]0;my title\x07\x1b]7;file://host/Users/me/a%20dir\x1b\\")
	assert.Equal(t, "my title", em.Title)
	assert.Equal(t, "/Users/me/a dir", em.Dir)
	assert.Equal(t, 2, nchanged)

	em.WriteString("\x1b]8;;https://x.org\x07link\x1b]8;;\x07 no")
	cs := em.Line(0).Cells
	assert.Equal(t, uint32(1), cs[0].Style.Link)
	assert.Equal(t, uint32(0), cs[5].Style.Link)
	assert.Equal(t, []string{"https://x.org"}, em.Links)

	var clip string
	em.Clipboard = func(s string) { clip = s }
	em.WriteString("\x1b]52;c;aGVsbG8=\x07")
	assert.Equal(t, "hello", clip)
}

func TestShellIntegration(t *testing.T) {
	em := NewEmulator(20, 5)
	em.WriteString("\x1b]133;A\x07$ \x1b]133;B\x07ls -l\r\n\x1b]133;C\x07out\r\n\x1b]133;D;2\x07")
	em.WriteString("\x1b]133;A\x07$ \x1b]133;B\x07")
	assert.Equal(t, 2, len(em.Commands))
	c := em.Commands[0]
	assert.Equal(t, "ls -l", c.Text)
	assert.Equal(t, 0, c.PromptLine)
	assert.Equal(t, 0, c.InputLine)
	assert.Equal(t, 2, c.InputCol)
	assert.Equal(t, 1, c.OutputLine)
	assert.Equal(t, 2, c.EndLine)
	assert.Equal(t, 2, c.ExitCode)
	assert.True(t, c.Done)
	assert.False(t, em.Commands[1].Done)

	em.WriteString("\x1b]633;E;echo a\\x3bb\x07\x1b]633;P;Cwd=/tmp\x07\x1b]633;C\x07")
	c = em.Commands[1]
	assert.Equal(t, "echo a;b", c.Text)
	assert.Equal(t, "/tmp", em.Dir)
	assert.Equal(t, "/tmp", c.Dir)
}

func TestResize(t *testing.T) {
	em := NewEmulator(6, 4)
	em.WriteString("abcdefghij\r\nxy\r\n$ ")
	assert.Equal(t, []string{"abcdef", "ghij", "xy", "$"}, screenLines(em))
	assert.Equal(t, [2]int{3, 2}, cursor(em))

	em.Resize(10, 4)
	assert.Equal(t, []string{"abcdefghij", "xy", "$", ""}, screenLines(em))
	assert.Equal(t, [2]int{2, 2}, cursor(em))

	em.Resize(4, 4)
	assert.Equal(t, []string{"efgh", "ij", "xy", "$"}, screenLines(em))
	sc, _ := em.TakeScrolled()
	assert.Equal(t, []string{"abcd"}, lineStrings(sc), "the first line goes to the scrollback")
	assert.True(t, sc[0].Wrapped)
	assert.Equal(t, [2]int{3, 2}, cursor(em))

	em.Resize(4, 2)
	assert.Equal(t, []string{"xy", "$"}, screenLines(em))
	assert.Equal(t, [2]int{1, 2}, cursor(em))
	em.Resize(4, 3)
	assert.Equal(t, []string{"xy", "$", ""}, screenLines(em))
	em.WriteString("\x1b[3;1Hz")
	em.Resize(4, 2) // fewer rows: blank lines below the cursor go first
	assert.Equal(t, []string{"$", "z"}, screenLines(em))
	em.WriteString("\x1b[1;1H")
	em.Resize(4, 1)
	assert.Equal(t, []string{"$"}, screenLines(em), "the cursor line is kept")
	em.Resize(4, 3)

	// the alternate screen is cut
	em.WriteString("\x1b[?1049h\x1b[Habcd")
	em.Resize(2, 3)
	assert.Equal(t, "ab", em.Line(0).String())
	em.WriteString("\x1b[?1049l")
	assert.Equal(t, []string{"$", "", ""}, screenLines(em))
	assert.Equal(t, [2]int{0, 0}, cursor(em))
}

func TestClearScrollback(t *testing.T) {
	em := NewEmulator(4, 4)
	em.WriteString("a\r\nb\r\nc\r\nd\r\n$ 1234")
	em.ClearScrollback()
	sc, cleared := em.TakeScrolled()
	assert.True(t, cleared)
	assert.Equal(t, 0, len(sc))
	assert.Equal(t, []string{"$ 12", "34", "", ""}, screenLines(em), "the wrapped cursor line is kept")
	assert.Equal(t, [2]int{1, 2}, cursor(em))
}
