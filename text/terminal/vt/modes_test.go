// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSGRAttrs tests setting and resetting each text attribute, the
// styles of underlines, and the skipped underline color.
func TestSGRAttrs(t *testing.T) {
	em := NewEmulator(20, 2)
	em.WriteString("\x1b[2;3;5;8;9;53;21ma")
	all := Faint | Italic | Blink | Invisible | Strikethrough | Overline | DoubleUnderline
	assert.Equal(t, all, em.Line(0).Cells[0].Style.Attrs)
	em.WriteString("\x1b[22;23;25;28;29;55;24mb")
	assert.Equal(t, Attrs(0), em.Line(0).Cells[1].Style.Attrs)
	em.WriteString("\x1b[4:2mc\x1b[4:0md\x1b[4:3me\x1b[58;2;1;2;3;1mf\x1b[58:5:3mg\x1b[mh")
	cs := em.Line(0).Cells
	assert.Equal(t, DoubleUnderline, cs[2].Style.Attrs, "double")
	assert.Equal(t, Attrs(0), cs[3].Style.Attrs, "none")
	assert.Equal(t, Underline, cs[4].Style.Attrs, "curly is shown as single")
	assert.Equal(t, Underline|Bold, cs[5].Style.Attrs, "the underline color is skipped")
	assert.Equal(t, Underline|Bold, cs[6].Style.Attrs)
	assert.Equal(t, Style{}, cs[7].Style, "reset with no parameters")
	em.WriteString("\x1b[38;5mi\x1b[38;2;1;2mj")
	assert.True(t, em.Line(0).Cells[8].Style.Fg.IsDefault(), "incomplete colors")

	assert.Equal(t, "default", ColorDefault.String())
	assert.Equal(t, "200", IndexedColor(200).String())
	assert.Equal(t, "#0a141e", RGBColor(10, 20, 30).String())
	assert.False(t, IndexedColor(1).IsDefault())
}

// TestPrivateModes tests the DEC private modes that are not covered
// elsewhere, and the private erase and cursor report sequences.
func TestPrivateModes(t *testing.T) {
	em := NewEmulator(10, 4)
	var replies []string
	em.Reply = func(b []byte) { replies = append(replies, string(b)) }
	em.WriteString("\x1b[?5;9;1004;1007h")
	md := em.Modes
	assert.True(t, md.ReverseVideo)
	assert.Equal(t, MouseX10, md.Mouse)
	assert.True(t, md.FocusEvents)
	assert.True(t, md.AltScroll)
	em.WriteString("\x1b[?1003h")
	assert.Equal(t, MouseAny, em.Modes.Mouse)
	em.WriteString("\x1b[?7l0123456789ab")
	assert.Equal(t, "012345678b", em.Line(0).String(), "no wrapping")
	em.WriteString("\x1b[?7h")

	// origin mode moves to the top of the scroll region
	em.WriteString("\x1b[2;3r\x1b[?6hX\x1b[?6l")
	assert.Equal(t, "X", em.Line(1).String())

	// the cursor saved with 1048
	em.WriteString("\x1b[3;4H\x1b[?1048h\x1b[H\x1b[?1048l")
	assert.Equal(t, [2]int{2, 3}, cursor(em))
	em.WriteString("\x1b[?6n")
	assert.Equal(t, []string{"\x1b[?3;4R"}, replies)

	// the selective erases erase as the normal ones
	em.WriteString("\x1b[Habc\x1b[1;2H\x1b[?K")
	assert.Equal(t, "a", em.Line(0).String())
	em.WriteString("\x1b[?2J")
	assert.Equal(t, "\n\n\n\n", em.String())

	// the alternate screen without saving the cursor (47)
	em.WriteString("\x1b[2;2Hp\x1b[?47hq\x1b[?47l")
	assert.False(t, em.IsAltScreen())
	assert.Equal(t, " p", em.Line(1).String())

	// 132 columns are not supported, but the screen is cleared
	em.WriteString("\x1b[?3h")
	assert.Equal(t, "\n\n\n\n", em.String())
	assert.Equal(t, [2]int{0, 0}, cursor(em))
}

// TestPrimaryScreen tests the lines and cursor of the primary screen
// while the alternate screen is shown.
func TestPrimaryScreen(t *testing.T) {
	em := NewEmulator(10, 3)
	em.WriteString("one\r\ntwo")
	assert.Equal(t, cursor(em), [2]int{em.PrimaryCursor().Row, em.PrimaryCursor().Col})
	em.WriteString("\x1b[?1049h\x1b[3;5Hvi")
	assert.Equal(t, []string{"one", "two", ""}, lineStrings(em.PrimaryLines()))
	pc := em.PrimaryCursor()
	assert.Equal(t, [2]int{1, 3}, [2]int{pc.Row, pc.Col})
	assert.Equal(t, "\n\n    vi\n", em.String())
}

// TestSoftResetBackTab tests the soft reset (DECSTR) and moving back to
// tab stops (CBT).
func TestSoftResetBackTab(t *testing.T) {
	em := NewEmulator(30, 4)
	em.WriteString("\x1b[?25l\x1b[4h\x1b[?7l\x1b[?1h\x1b=\x1b[2;3r\x1b[1m")
	em.WriteString("\x1b[!p")
	md := em.Modes
	assert.False(t, md.CursorHidden)
	assert.False(t, md.Insert)
	assert.True(t, md.AutoWrap)
	assert.False(t, md.AppCursorKeys)
	assert.False(t, md.AppKeypad)
	assert.Equal(t, Style{}, em.Cursor.Style)

	em.WriteString("\x1b[1;20H\x1b[Z")
	assert.Equal(t, 16, em.Cursor.Col)
	em.WriteString("\x1b[2Z")
	assert.Equal(t, 0, em.Cursor.Col)
	em.WriteString("\x1b[Z")
	assert.Equal(t, 0, em.Cursor.Col, "at the start")
}
