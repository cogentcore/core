// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/terminal/vt"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readInputCmd returns a shell command that sets the given modes, then
// reads n bytes of raw input, and prints them in hex after "got:" on the
// primary screen, after "ready" shows that it is reading.
func readInputCmd(modes, reset string, n int) string {
	return fmt.Sprintf(`printf '%s'; stty raw -echo; printf 'ready'; v=$(dd bs=1 count=%d 2>/dev/null | od -An -tx1 | tr -d ' \n'); stty sane; printf '%s'; echo; echo "got:$v"`, modes, n, reset)
}

// waitGot waits for the n-th "got:" line of [readInputCmd] (from 1) and
// returns its hex.
func waitGot(t *testing.T, b *core.Body, tm *Terminal, n int) string {
	t.Helper()
	got := ""
	waitFor(t, b, "the input", func() bool {
		i := 0
		for _, ln := range strings.Split(linesText(tm), "\n") {
			if s, ok := strings.CutPrefix(ln, "got:"); ok && s != "$v\"" {
				if i++; i < n {
					continue
				}
				got = s
				return true
			}
		}
		return false
	})
	return got
}

// TestTerminalMouseReport tests reporting mouse presses, releases and
// wheel scrolling in the alternate screen to a program that asks for
// them, and the arrow keys sent for the wheel in alternate scroll mode.
func TestTerminalMouseReport(t *testing.T) {
	b, tm := testTerminal(t)
	b.AsyncLock()
	tm.Run(readInputCmd(`\033[?1049h\033[?1000h\033[?1006h`, `\033[?1006l\033[?1000l\033[?1049l`, 18))
	b.AsyncUnlock()
	waitFor(t, b, "the reading program", func() bool {
		return tm.altShown && strings.Contains(tm.gridView.Lines.String(), "ready")
	})
	b.AsyncLock()
	assert.Same(t, tm.gridView, tm.GridView())
	assert.Same(t, tm.linesView, tm.LinesView())
	assert.NotNil(t, tm.Emulator())
	gv := tm.gridView
	cs := gv.CharSize()
	pos := gv.Geom.ContentBBox.Min.Add(image.Pt(int(2.5*cs.X), int(1.5*cs.Y)))
	b.AsyncUnlock()
	b.AsyncLock()
	for _, typ := range []events.Types{events.MouseDown, events.MouseUp} {
		e := events.NewMouse(typ, events.Left, pos, 0)
		e.SetLocalOff(image.Point{}) // as the events of the window
		gv.HandleEvent(e)
	}
	b.AsyncUnlock()
	got := waitGot(t, b, tm, 1)
	// SGR reports of the press and release at column 3, row 2
	press, release := "\x1b[<0;3;2M", "\x1b[<0;3;2m"
	assert.Equal(t, fmt.Sprintf("%x%x", press, release), got)

	// the wheel, in alternate scroll mode, sends arrow keys
	b.AsyncLock()
	tm.Run(readInputCmd(`\033[?1049h\033[?1007h`, `\033[?1007l\033[?1049l`, 3))
	b.AsyncUnlock()
	waitFor(t, b, "the reading program", func() bool {
		return tm.altShown && strings.Contains(tm.gridView.Lines.String(), "ready")
	})
	b.AsyncLock()
	assert.Equal(t, vt.MouseNone, tm.modes().Mouse)
	se := events.NewScroll(pos, math32.Vec2(0, 3*cs.Y), 0)
	se.SetLocalOff(image.Point{})
	gv.HandleEvent(se)
	b.AsyncUnlock()
	got = waitGot(t, b, tm, 2)
	assert.Equal(t, fmt.Sprintf("%x", "\x1b[B"), got[:6], "the down arrow")
}

// TestTerminalFocusReport tests reporting focus changes to a program
// that asks for them.
func TestTerminalFocusReport(t *testing.T) {
	b, tm := testTerminal(t)
	b.AsyncLock()
	tm.Run(readInputCmd(`\033[?1004h`, `\033[?1004l`, 6))
	b.AsyncUnlock()
	waitFor(t, b, "the reading program", func() bool {
		return strings.Contains(linesText(tm), "ready")
	})
	b.AsyncLock()
	tm.focusChanged(false)
	tm.focusChanged(true)
	b.AsyncUnlock()
	assert.Equal(t, fmt.Sprintf("%x", "\x1b[O\x1b[I"), waitGot(t, b, tm, 1))
}

// TestTerminalTitleShortcuts tests the title set by the program, copying
// the selection and pasting the clipboard with the keys of the GUI,
// scrolling the lines with Shift and the page keys, and restarting.
func TestTerminalTitleShortcuts(t *testing.T) {
	b, tm := testTerminal(t)
	titles := make(chan string, 4)
	tm.OnTitle(func(title string) { titles <- title })
	b.AsyncLock()
	tm.Run(`printf '\033]0;My Title\007'; echo copyme`)
	b.AsyncUnlock()
	waitFor(t, b, "the output", func() bool { return hasOutputLine(tm, "copyme") })
	assert.Equal(t, "My Title", tm.Title())
	assert.Equal(t, "My Title", <-titles)

	b.AsyncLock()
	lv := tm.linesView
	tm.SetFocus()
	lv.selectAll()
	sendShortcut(lv, keymap.Copy)
	md := lv.Clipboard().Read([]string{mimedata.TextPlain})
	require.NotNil(t, md)
	assert.Contains(t, md.Text(mimedata.TextPlain), "copyme")
	lv.SelectReset()
	lv.Clipboard().Write(mimedata.NewText("echo pasted"))
	sendShortcut(lv, keymap.Paste)
	tm.Run("")
	for _, code := range []key.Codes{key.CodePageUp, key.CodePageDown, key.CodeHome, key.CodeEnd} {
		core.SendKey(lv, 0, code, key.Shift)
	}
	b.AsyncUnlock()
	waitFor(t, b, "the pasted command", func() bool { return hasOutputLine(tm, "pasted") })

	b.AsyncLock()
	require.NoError(t, tm.Restart())
	b.AsyncUnlock()
	waitFor(t, b, "the restarted shell", func() bool { return tm.IsRunning() })
}

// sendShortcut sends the key chord of the given key function of the
// terminal GUI (see [ShortcutChord]) to the given view.
func sendShortcut(vw *View, kf keymap.Functions) {
	rn, code, mods, err := ShortcutChord(kf).Decode()
	if err != nil {
		panic(err)
	}
	vw.HandleEvent(events.NewKey(events.KeyChord, rn, code, mods))
}

// TestControlChar tests the control characters of keys typed with Control.
func TestControlChar(t *testing.T) {
	for rn, want := range map[rune]rune{'a': 1, 'z': 26, 'A': 1, '[': 0x1b, '_': 0x1f, ' ': 0, '2': 0, '3': 0x1b, '7': 0x1f, '8': 0x7f, '?': 0x7f, '/': 0x1f} {
		got, ok := controlChar(rn)
		assert.True(t, ok, string(rn))
		assert.Equal(t, want, got, string(rn))
	}
	_, ok := controlChar('1')
	assert.False(t, ok)
}

// TestShiftUserInput tests moving the positions of user input when lines
// are removed from the start.
func TestShiftUserInput(t *testing.T) {
	tm := &Terminal{}
	tm.inputStart = textpos.Pos{Line: 5}
	tm.entered = []textpos.Pos{{Line: 1}, {Line: 4, Char: 2}}
	tm.shiftUserInput(3)
	assert.Equal(t, 2, tm.inputStart.Line)
	assert.Equal(t, []textpos.Pos{{Line: 1, Char: 2}}, tm.entered)
	tm.shiftUserInput(5)
	assert.Equal(t, -1, tm.inputStart.Line)
	assert.Empty(t, tm.entered)
}

// TestLink tests the hyperlink of the markup of a cell.
func TestLink(t *testing.T) {
	cm := &colorMap{links: []string{"https://example.com"}}
	rs := rich.NewStyle()
	cm.link(rs, vt.Style{Link: 1})
	assert.Equal(t, "https://example.com", rs.URL)
	assert.Equal(t, rich.Link, rs.Special)
	rs = rich.NewStyle()
	cm.link(rs, vt.Style{Link: 2})
	assert.Equal(t, "", rs.URL, "not a link")
}
