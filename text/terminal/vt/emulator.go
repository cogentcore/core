// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vt implements a virtual terminal emulator, [Emulator]:
// the grid of character
// cells that results from the output of programs that use the standard
// (VT100 / xterm) terminal control sequences, with the scrolling
// scrollback, modes, and other state of the terminal. It has no GUI:
// see the parent terminal package for the widget that shows it.
// The control sequences are parsed with github.com/charmbracelet/x/ansi.
package vt

import (
	"image/color"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
)

// Emulator emulates a terminal: it has the state of the terminal,
// mainly the cell grid of the current screen, which is updated from the
// output of a program, given to [Emulator.Write]. Replies to queries from the program, such as for the
// cursor position, are sent to [Emulator.Reply]. The methods do not
// lock the terminal: users that access it from multiple goroutines
// must use its mutex.
type Emulator struct {
	// Cols is the number of columns (cells per line).
	Cols int

	// Rows is the number of rows (lines) in the screen.
	Rows int

	// Cursor is the cursor position and the state that goes with it.
	Cursor Cursor

	// Modes are the terminal modes set by the program.
	Modes Modes

	// Title is the window title, set by the program.
	Title string

	// Dir is the current working directory of the program,
	// as reported by it with the OSC 7 sequence (as shells do with
	// shell integration).
	Dir string

	// Links are the URLs of hyperlinks (OSC 8), indexed by [Style.Link]-1.
	Links []string

	// Commands are the commands that have been run, as reported by
	// the shell with shell integration sequences (OSC 133 or OSC 633).
	Commands []*Command

	// MaxCommands is the maximum number of [Emulator.Commands] to keep.
	MaxCommands int

	// Foreground and Background are the default colors, which are reported
	// to programs that query them (OSC 10 and 11), such as editors that
	// choose a light or dark theme.
	Foreground, Background color.RGBA

	// Reply is called with the replies to queries from the program,
	// which must be written to its input.
	Reply func(b []byte)

	// Bell is called for the bell control character.
	Bell func()

	// Changed is called when the title, the working directory, or the
	// commands have changed.
	Changed func()

	// Clipboard is called when the program sets the clipboard (OSC 52).
	// It is off unless set, for security.
	Clipboard func(text string)

	// primary is the main screen, which has the scrollback.
	primary *screen

	// alt is the alternate screen used by full-screen programs.
	alt *screen

	// screen is the current screen, primary or alt.
	screen *screen

	// scrolled are the lines scrolled off the top of the primary screen
	// since the last [Emulator.TakeScrolled].
	scrolled []*Line

	// numScrolled is the total number of lines scrolled off the top of
	// the primary screen, for the absolute line numbers of [Command]s.
	numScrolled int

	// scrollbackCleared is set when the program clears the scrollback (ED 3).
	scrollbackCleared bool

	// top and bottom are the scroll region rows (bottom is inclusive).
	top, bottom int

	// tabStops are the tab stop columns.
	tabStops []bool

	// lastRune is the last printed character, for the REP sequence.
	lastRune rune

	// linkURL is the current hyperlink index (1-based), or 0.
	link uint32

	// parser is the control sequence parser.
	parser *ansi.Parser

	// Mutex is for users of the Emulator from multiple goroutines.
	sync.Mutex
}

// Cursor is the cursor position and the state that is saved and
// restored with it.
type Cursor struct {
	// Row and Col are the cursor position.
	Row, Col int

	// Style is the style for newly written characters.
	Style Style

	// pendingWrap is set when a character is written in the last
	// column, so the next one wraps to the next line.
	pendingWrap bool

	// charsets are the G0 and G1 character sets.
	charsets [2]charset

	// shift is the active character set (0 or 1).
	shift int

	// originMode is the DECOM mode, which is saved with the cursor.
	originMode bool
}

// charset is a character set designation.
type charset uint8

const (
	charsetASCII charset = iota
	charsetDECGraphics
)

// Modes are the terminal modes that programs set.
type Modes struct {
	// AutoWrap wraps text at the end of a line (DECAWM). On by default.
	AutoWrap bool

	// Insert inserts characters rather than replacing them (IRM).
	Insert bool

	// NewLine makes a line feed also do a carriage return (LNM).
	NewLine bool

	// CursorHidden is set when the program hides the cursor (DECTCEM).
	CursorHidden bool

	// AppCursorKeys makes the cursor keys send application codes (DECCKM).
	AppCursorKeys bool

	// AppKeypad makes the keypad send application codes (DECKPAM).
	AppKeypad bool

	// ReverseVideo swaps the default colors of the screen (DECSCNM).
	ReverseVideo bool

	// BracketedPaste brackets pasted text with escape codes.
	BracketedPaste bool

	// FocusEvents sends codes when the terminal gains or loses focus.
	FocusEvents bool

	// Mouse is the mouse tracking mode.
	Mouse MouseModes

	// MouseSGR uses the SGR encoding for mouse events (mode 1006).
	MouseSGR bool

	// AltScroll sends cursor keys for the mouse wheel in the alternate
	// screen (mode 1007). On by default.
	AltScroll bool

	// CursorShape is the cursor shape set by the program (DECSCUSR):
	// 0-1 blinking block, 2 block, 3-4 underline, 5-6 bar.
	CursorShape int
}

// MouseModes are the mouse tracking modes.
type MouseModes uint8

const (
	// MouseNone does no mouse tracking.
	MouseNone MouseModes = iota

	// MouseX10 reports button presses (mode 9).
	MouseX10

	// MouseNormal reports button presses and releases (mode 1000).
	MouseNormal

	// MouseButton also reports motion with a button down (mode 1002).
	MouseButton

	// MouseAny reports all motion (mode 1003).
	MouseAny
)

// NewEmulator returns a new terminal with the given size.
func NewEmulator(cols, rows int) *Emulator {
	em := &Emulator{MaxCommands: 1000}
	em.parser = ansi.NewParser()
	em.parser.SetHandler(ansi.Handler{
		Print:     em.print,
		Execute:   em.execute,
		HandleCsi: em.handleCSI,
		HandleEsc: em.handleESC,
		HandleOsc: em.handleOSC,
	})
	em.Foreground = color.RGBA{0, 0, 0, 255}
	em.Background = color.RGBA{255, 255, 255, 255}
	em.Cols = max(cols, 1)
	em.Rows = max(rows, 1)
	em.primary = newScreen(em.Cols, em.Rows)
	em.alt = newScreen(em.Cols, em.Rows)
	em.reset()
	return em
}

// reset sets the terminal to its initial state, except for the scrollback.
func (em *Emulator) reset() {
	em.screen = em.primary
	em.primary.clear()
	em.alt.clear()
	em.Cursor = Cursor{}
	em.primary.saved = Cursor{}
	em.alt.saved = Cursor{}
	em.Modes = Modes{AutoWrap: true, AltScroll: true}
	em.top = 0
	em.bottom = em.Rows - 1
	em.resetTabStops()
	em.link = 0
	em.Links = nil
}

// Write processes the given output of the program.
// It always returns len(b), nil.
func (em *Emulator) Write(b []byte) (int, error) {
	em.parser.Parse(b)
	return len(b), nil
}

// WriteString processes the given output of the program.
func (em *Emulator) WriteString(s string) {
	em.Write([]byte(s))
}

// IsAltScreen returns whether the alternate screen is active,
// as it is for full-screen programs such as editors.
func (em *Emulator) IsAltScreen() bool {
	return em.screen == em.alt
}

// Lines returns the lines of the current screen. They must not be
// changed, and are only valid until the next Write.
func (em *Emulator) Lines() []*Line {
	return em.screen.lines
}

// PrimaryLines returns the lines of the primary screen, even when the
// alternate screen is active. They must not be changed, and are only
// valid until the next Write.
func (em *Emulator) PrimaryLines() []*Line {
	return em.primary.lines
}

// PrimaryCursor returns the cursor of the primary screen, which is
// saved while the alternate screen is active.
func (em *Emulator) PrimaryCursor() Cursor {
	if em.IsAltScreen() {
		return em.primary.saved
	}
	return em.Cursor
}

// ClearScrollback clears the scrollback and the lines of the primary screen
// above the cursor line, which becomes the first line, as for a clear
// command in the terminal GUI that keeps the current prompt.
func (em *Emulator) ClearScrollback() {
	em.scrolled = nil
	em.scrollbackCleared = true
	if em.IsAltScreen() {
		return
	}
	row := em.Cursor.Row
	for row > 0 && em.primary.lines[row-1].Wrapped {
		row--
	}
	if row > 0 {
		top, bottom := em.top, em.bottom
		em.top, em.bottom = 0, em.Rows-1
		em.scrollUpRegion(0, row, false)
		em.top, em.bottom = top, bottom
		em.Cursor.Row -= row
	}
}

// Line returns the given row of the current screen.
func (em *Emulator) Line(row int) *Line {
	return em.screen.lines[row]
}

// String returns the text of the current screen, without the trailing
// blank cells of each line, with a line feed after each line.
func (em *Emulator) String() string {
	var sb strings.Builder
	for _, ln := range em.screen.lines {
		sb.WriteString(ln.String())
		sb.WriteByte('\n')
	}
	return sb.String()
}

// TakeScrolled returns the lines that have scrolled off the top of the
// primary screen since the last call, for the scrollback,
// and whether the scrollback was cleared (ED 3) before them.
func (em *Emulator) TakeScrolled() (lines []*Line, cleared bool) {
	lines, cleared = em.scrolled, em.scrollbackCleared
	em.scrolled = nil
	em.scrollbackCleared = false
	return
}

// NumScrolled returns the total number of lines that have scrolled
// off the top of the primary screen. The absolute line number of a
// row in the screen is NumScrolled + row.
func (em *Emulator) NumScrolled() int {
	return em.numScrolled
}

// reply sends the given reply to the program.
func (em *Emulator) reply(s string) {
	if em.Reply != nil {
		em.Reply([]byte(s))
	}
}

// changed calls the Changed function.
func (em *Emulator) changed() {
	if em.Changed != nil {
		em.Changed()
	}
}

// resetTabStops sets tab stops every 8 columns.
func (em *Emulator) resetTabStops() {
	em.tabStops = make([]bool, em.Cols)
	for i := 8; i < em.Cols; i += 8 {
		em.tabStops[i] = true
	}
}
