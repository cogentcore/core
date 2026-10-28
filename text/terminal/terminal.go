// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package terminal provides a terminal widget that runs a program,
// typically a shell, in a pseudo-terminal. The output of the program is
// shown in a lines view, which has the scrollback and the screen
// as lines of text that are wrapped to the width of the view, using
// [lines.Lines]. Full-screen programs such as editors use the alternate
// screen, which is shown in a grid view on top of the lines view. The
// terminal state is in the vt package and the pseudo-terminal in the
// pty package.
package terminal

//go:generate core generate

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/terminal/pty"
	"cogentcore.org/core/text/terminal/vt"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/tree"
)

// Terminal is a terminal widget that runs a program, typically a shell,
// in a pseudo-terminal. It starts the program when it is first shown,
// unless it has been started already with [Terminal.Start]. It has a
// lines view of the scrollback and the screen, and a grid view of the
// alternate screen used by full-screen programs, in a stack, with the
// one in use on top. The context menus added to the terminal with
// [core.WidgetBase.AddContextMenu] are shown in the context menu of
// the views, before the items for copying, pasting, and clearing.
type Terminal struct {
	core.Frame

	// Command is the program to run, with its arguments.
	// The default is the user's shell (see [pty.DefaultShell]).
	Command []string

	// Dir is the directory to start the program in.
	// The default is the current directory.
	Dir string

	// Env are environment variables to add to the environment
	// of the program, as "key=value". The standard variables for
	// the terminal type (TERM etc) are added automatically.
	Env []string

	// MaxScrollback is the maximum number of lines of scrollback to keep.
	MaxScrollback int

	// AltAsMeta makes the Alt key (Option on macOS) with a character
	// send Escape and the character, as the Meta key, which is used by
	// shells for word movement etc, rather than the special characters
	// that Option makes on macOS.
	AltAsMeta bool

	// emulator is the terminal emulator, with the terminal state.
	emulator *vt.Emulator

	// pty is the pseudo-terminal of the running program.
	pty *pty.Pty

	// linesView is the view of the scrollback and the primary screen.
	linesView *View

	// gridView is the view of the alternate screen.
	gridView *View

	// numScrollback is the number of lines of scrollback in the lines view,
	// after which the lines of the screen are.
	numScrollback int

	// pending are the lines that have scrolled into the scrollback whose
	// text continues on the next line (wrapped), which are shown with
	// the screen until the end of their text has scrolled too.
	pending []*vt.Line

	// size is the size of the terminal, from the lines view.
	size image.Point

	// altShown is whether the alternate screen is shown.
	altShown bool

	// toEnd is set when the lines view should scroll to the end
	// at the next update, after input from the user.
	toEnd bool

	// syncScheduled is set when an update of the views is scheduled.
	syncScheduled atomic.Bool

	// infoChanged is set when the title, directory, or commands changed.
	infoChanged atomic.Bool

	// exited is set when the program has exited.
	exited atomic.Bool

	// exitErr is the error of the exited program.
	exitErr error

	// title and dir are the last title and directory sent to listeners.
	title, dir string

	// input is the queue of input to send to the running program.
	input atomic.Pointer[inputQueue]

	titleFuncs, dirFuncs []func(s string)
	exitFuncs            []func(err error)
	commandFuncs         []func(c *vt.Command)

	// numCommands is the number of commands reported to listeners.
	numCommands int

	// exitSent is whether the exit has been sent to listeners.
	exitSent bool

	// linesCursor is the position of the terminal cursor in the lines view,
	// with a Line of -1 if it is not there.
	linesCursor textpos.Pos
}

func (tm *Terminal) Init() {
	tm.Frame.Init()
	tm.MaxScrollback = 10000
	tm.AltAsMeta = true
	tm.emulator = vt.NewEmulator(80, 24)
	tm.emulator.Reply = func(b []byte) { tm.send(b, false) }
	tm.emulator.Changed = func() {
		tm.infoChanged.Store(true)
	}
	tm.emulator.Clipboard = func(text string) {
		// not with the terminal locked, as sync has the render lock first
		go func() {
			if tm.AsyncLockIfShown() {
				tm.Clipboard().Write(mimedata.NewText(text))
				tm.AsyncUnlock()
			}
		}()
	}
	tm.Styler(func(s *styles.Style) {
		s.Display = styles.Stacked
		s.Grow.Set(1, 1)
		s.Min.X.Ch(20)
		s.Min.Y.Em(4)
		s.Padding.Zero()
		s.Gap.Zero()
		s.Font.Family = rich.Monospace
	})
	tm.OnShow(func(e events.Event) {
		tm.sync()
	})
	tree.AddChildAt(tm, "lines", func(w *View) {
		tm.linesView = w
		w.Terminal = tm
		tm.configLines(w)
	})
	tree.AddChildAt(tm, "grid", func(w *View) {
		tm.gridView = w
		w.Terminal = tm
		w.isGrid = true
		tm.configLines(w)
	})
}

// Destroy stops the program, if it is running.
func (tm *Terminal) Destroy() {
	tm.Stop()
	tm.Frame.Destroy()
}

// configLines configures the lines of the given view.
func (tm *Terminal) configLines(vw *View) {
	ln := lines.NewLines()
	ln.SetReadOnly(true)
	ln.SetUndoOn(false)
	vw.SetLines(ln)
	vw.Styler(func(s *styles.Style) {
		ln.Settings.LineNumbers = false // the editor sets it from the settings
	})
}

// Emulator returns the terminal emulator, with the terminal state,
// which must be locked to use it.
func (tm *Terminal) Emulator() *vt.Emulator {
	return tm.emulator
}

// LinesView returns the view of the scrollback and the primary screen.
func (tm *Terminal) LinesView() *View {
	return tm.linesView
}

// GridView returns the view of the alternate screen.
func (tm *Terminal) GridView() *View {
	return tm.gridView
}

// ActiveView returns the view that is shown: the grid view when a
// program uses the alternate screen, and otherwise the lines view.
func (tm *Terminal) ActiveView() *View {
	if tm.altShown {
		return tm.gridView
	}
	return tm.linesView
}

// SetFocus sets the focus to the view that is shown.
func (tm *Terminal) SetFocus() {
	if vw := tm.ActiveView(); vw != nil {
		vw.SetFocus()
		return
	}
	tm.Frame.SetFocus()
}

// IsRunning returns whether the program is running.
func (tm *Terminal) IsRunning() bool {
	return tm.pty != nil && !tm.exited.Load()
}

// Start starts the program, if it is not already running.
// It is called automatically when the terminal is first shown.
func (tm *Terminal) Start() error {
	if tm.IsRunning() {
		return nil
	}
	args := tm.Command
	if len(args) == 0 {
		args = pty.DefaultShell()
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = tm.Dir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor", "TERM_PROGRAM=CogentCore")
	cmd.Env = append(cmd.Env, tm.Env...)
	cols, rows := 80, 24
	if tm.size.X > 0 && tm.size.Y > 0 {
		cols, rows = tm.size.X, tm.size.Y
	}
	tm.emulator.Lock()
	tm.emulator.Resize(cols, rows)
	tm.emulator.Unlock()
	p, err := pty.Start(cmd, cols, rows)
	if err != nil {
		tm.emulator.Lock()
		tm.emulator.WriteString(fmt.Sprintf("\r\n[could not start %s: %v]\r\n", strings.Join(args, " "), err))
		tm.emulator.Unlock()
		tm.scheduleSync()
		return err
	}
	tm.emulator.Lock()
	tm.pty = p
	tm.emulator.Unlock()
	tm.exitSent = false
	tm.exited.Store(false)
	tm.exitErr = nil
	tm.input.Store(newInputQueue(p))
	go tm.readOutput(p)
	return nil
}

// Stop ends the program, if it is running, by closing the pseudo-terminal,
// which sends it a hangup signal.
func (tm *Terminal) Stop() {
	if tm.pty == nil {
		return
	}
	tm.pty.Close()
	if tm.pty.Cmd.Process != nil && runtime.GOOS != "windows" {
		go func(p *os.Process) {
			time.Sleep(time.Second)
			p.Kill() // in case it ignores the hangup
		}(tm.pty.Cmd.Process)
	}
}

// Restart stops the program if it is running, and starts it again.
func (tm *Terminal) Restart() error {
	tm.Stop()
	tm.emulator.Lock()
	tm.pty = nil
	tm.emulator.Unlock()
	return tm.Start()
}

// restartOnKey restarts the program after it has exited,
// returning true if it does.
func (tm *Terminal) restartOnKey() bool {
	if tm.pty == nil || !tm.exited.Load() {
		return false
	}
	tm.emulator.Lock()
	tm.pty = nil
	tm.emulator.WriteString("\r\n")
	tm.emulator.Unlock()
	tm.Start()
	return true
}

// readOutput reads the output of the program until it exits.
func (tm *Terminal) readOutput(p *pty.Pty) {
	buf := make([]byte, 64*1024)
	for {
		n, err := p.Read(buf)
		if n > 0 {
			tm.emulator.Lock()
			tm.emulator.Write(buf[:n])
			tm.emulator.Unlock()
			tm.scheduleSync()
		}
		if err != nil {
			break
		}
	}
	err := p.Wait()
	p.Close()
	tm.emulator.Lock()
	defer tm.emulator.Unlock()
	if tm.pty != p { // restarted
		return
	}
	if iq := tm.input.Swap(nil); iq != nil {
		iq.stop()
	}
	msg := "[process exited]"
	if err != nil {
		msg = fmt.Sprintf("[process exited: %v]", err)
	}
	tm.emulator.WriteString("\x1b[?1049l\x1b[0m\r\n" + msg + " (press a key to restart)")
	tm.exitErr = err
	tm.exited.Store(true)
	tm.infoChanged.Store(true)
	tm.scheduleSync()
}

// send sends the given input to the program. If user is set,
// it is input from the user, after which the lines view scrolls
// to the end.
func (tm *Terminal) send(b []byte, user bool) {
	if len(b) == 0 {
		return
	}
	if user {
		tm.toEnd = true
		if tm.linesView != nil && !tm.altShown {
			tm.linesView.SetCursorTarget(textpos.PosErr)
		}
	}
	if iq := tm.input.Load(); iq != nil {
		iq.send(b)
	}
}

// SendText sends the given text to the program as input,
// as if it was typed, without any processing.
func (tm *Terminal) SendText(s string) {
	tm.send([]byte(s), true)
}

// Paste sends the given text to the program as pasted text: line
// breaks are sent as carriage returns (as for the Return key), and the
// text is bracketed if the program asks for it, so that, for example,
// shells do not run the lines in it until Return is pressed.
func (tm *Terminal) Paste(s string) {
	s = strings.ReplaceAll(s, "\r\n", "\r")
	s = strings.ReplaceAll(s, "\n", "\r")
	md := tm.modes()
	if md.BracketedPaste {
		s = "\x1b[200~" + strings.ReplaceAll(s, "\x1b[201~", "") + "\x1b[201~"
	}
	tm.send([]byte(s), true)
}

// PasteClipboard pastes the text in the clipboard.
func (tm *Terminal) PasteClipboard() {
	data := tm.Clipboard().Read([]string{fileinfo.TextPlain})
	if data == nil {
		return
	}
	tm.Paste(data.Text(fileinfo.TextPlain))
}

// Run runs the given command line, by sending it to the shell
// followed by Return. It is up to the user to only do this
// when the shell is waiting for a command.
func (tm *Terminal) Run(cmd string) {
	tm.SendText(cmd + "\r")
}

// Cd changes the directory of the shell to the given one,
// by running the cd command with it.
func (tm *Terminal) Cd(dir string) {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	tm.Run(cdCommand(tm.shellName(), dir))
}

// shellName returns the base name of the program, without any extension.
func (tm *Terminal) shellName() string {
	args := tm.Command
	if len(args) == 0 {
		args = pty.DefaultShell()
	}
	nm := strings.ToLower(filepath.Base(args[0]))
	return strings.TrimSuffix(nm, filepath.Ext(nm))
}

// cdCommand returns the command to change to the given directory in the
// given shell, with the directory quoted as needed.
func cdCommand(shell, dir string) string {
	switch shell {
	case "cmd":
		return `cd /d "` + dir + `"`
	case "powershell", "pwsh":
		return "cd '" + strings.ReplaceAll(dir, "'", "''") + "'"
	}
	return "cd '" + strings.ReplaceAll(dir, "'", `'\''`) + "'"
}

// ClearScrollback clears the scrollback and the screen, except for the line
// of the cursor, which is typically the prompt of the shell.
func (tm *Terminal) ClearScrollback() {
	tm.emulator.Lock()
	tm.emulator.ClearScrollback()
	tm.emulator.Unlock()
	tm.sync()
}

// WorkingDir returns the working directory of the shell, as it reports
// with shell integration (OSC 7), or the starting directory.
func (tm *Terminal) WorkingDir() string {
	tm.emulator.Lock()
	defer tm.emulator.Unlock()
	if tm.emulator.Dir != "" {
		return tm.emulator.Dir
	}
	return tm.Dir
}

// Title returns the title set by the program.
func (tm *Terminal) Title() string {
	tm.emulator.Lock()
	defer tm.emulator.Unlock()
	return tm.emulator.Title
}

// Commands returns copies of the commands that have been run, as reported
// by the shell with shell integration (OSC 133 or 633).
func (tm *Terminal) Commands() []vt.Command {
	tm.emulator.Lock()
	defer tm.emulator.Unlock()
	cs := make([]vt.Command, len(tm.emulator.Commands))
	for i, c := range tm.emulator.Commands {
		cs[i] = *c
	}
	return cs
}

// OnTitle adds a function to call when the title set by the program changes.
func (tm *Terminal) OnTitle(fun func(title string)) {
	tm.titleFuncs = append(tm.titleFuncs, fun)
}

// OnDir adds a function to call when the working directory of the shell
// changes, as reported with shell integration (OSC 7).
func (tm *Terminal) OnDir(fun func(dir string)) {
	tm.dirFuncs = append(tm.dirFuncs, fun)
}

// OnExit adds a function to call when the program exits.
func (tm *Terminal) OnExit(fun func(err error)) {
	tm.exitFuncs = append(tm.exitFuncs, fun)
}

// OnCommand adds a function to call when a command finishes, as reported
// by the shell with shell integration (OSC 133 or 633).
func (tm *Terminal) OnCommand(fun func(c *vt.Command)) {
	tm.commandFuncs = append(tm.commandFuncs, fun)
}

// modes returns the current terminal modes.
func (tm *Terminal) modes() vt.Modes {
	tm.emulator.Lock()
	defer tm.emulator.Unlock()
	return tm.emulator.Modes
}

// focusChanged reports a focus change to the program, if it asks for it.
func (tm *Terminal) focusChanged(focused bool) {
	if md := tm.modes(); md.FocusEvents {
		if focused {
			tm.send([]byte("\x1b[I"), false)
		} else {
			tm.send([]byte("\x1b[O"), false)
		}
	}
	tm.scheduleSync() // the cursor color
}

// viewSized is called when the lines view has been sized,
// with the number of columns and rows that it shows.
func (tm *Terminal) viewSized(sz image.Point) {
	if sz.X <= 0 || sz.Y <= 0 || sz == tm.size {
		return
	}
	tm.size = sz
	if tm.pty == nil {
		tm.Start()
		return
	}
	tm.emulator.Lock()
	tm.emulator.Resize(sz.X, sz.Y)
	tm.emulator.Unlock()
	if tm.IsRunning() {
		tm.pty.Resize(sz.X, sz.Y)
	}
	tm.sync()
}

// scheduleSync schedules an update of the views from the terminal state,
// shortly, so that output that comes in many pieces is shown at once.
func (tm *Terminal) scheduleSync() {
	if tm.syncScheduled.Swap(true) {
		return
	}
	time.AfterFunc(10*time.Millisecond, func() {
		tm.syncScheduled.Store(false)
		if !tm.AsyncLockIfShown() {
			return
		}
		tm.sync()
		tm.AsyncUnlock()
	})
}

// sync updates the views from the terminal state.
func (tm *Terminal) sync() {
	lv, gv := tm.linesView, tm.gridView
	if lv == nil || gv == nil || lv.Lines == nil || gv.Lines == nil {
		return
	}
	t := tm.emulator
	t.Lock()
	scrolled, cleared := t.TakeScrolled()
	alt := t.IsAltScreen()
	cm := newColorMap(lv.Lines.FontStyle(), slices.Clone(t.Links))
	cm.cursor, cm.cursorFocused = tm.cursorColor()
	t.Foreground, t.Background = cm.fg, cm.bg
	md := t.Modes
	cur := t.Cursor
	var gridText [][]rune
	var gridMarkup []rich.Text
	if alt {
		for r, ln := range t.Lines() {
			minLen, cursor := 0, -1
			if r == cur.Row {
				minLen = cur.Col + 1
				if !md.CursorHidden {
					cursor = cur.Col
				}
			}
			rs, mu := cm.lineMarkup(ln.Cells, minLen, cursor)
			gridText = append(gridText, rs)
			gridMarkup = append(gridMarkup, mu)
		}
	}
	var screen []*vt.Line
	pcur := t.PrimaryCursor()
	if !alt || len(scrolled) > 0 || cleared || tm.numScrollback == 0 {
		screen = slices.Clone(t.PrimaryLines())
		for i, ln := range screen {
			screen[i] = &vt.Line{Cells: slices.Clone(ln.Cells), Wrapped: ln.Wrapped}
		}
	}
	t.Unlock()

	toEnd := tm.toEnd || lv.IsScrolledToEnd()
	tm.toEnd = false
	if screen != nil {
		tm.syncLines(cm, scrolled, cleared, screen, pcur, !md.CursorHidden && !alt)
	}
	if alt {
		gv.Lines.ReplaceLinesMarkup(0, -1, gridText, gridMarkup)
		gv.NeedsRender()
	}
	if alt != tm.altShown {
		tm.altShown = alt
		focused := lv.StateIs(states.Focused) || gv.StateIs(states.Focused)
		tm.StackTop = 0
		if alt {
			tm.StackTop = 1
		}
		tm.NeedsLayout()
		if focused {
			tm.ActiveView().SetFocus()
		}
	}
	if toEnd && !alt {
		lv.SetCursorTarget(textpos.PosErr)
	}
	lv.NeedsRender()
	tm.sendInfo()
}

// syncLines updates the lines view from the given lines scrolled into
// the scrollback, and the lines of the primary screen, with its cursor,
// which is shown if showCursor is true.
func (tm *Terminal) syncLines(cm *colorMap, scrolled []*vt.Line, cleared bool, screen []*vt.Line, cur vt.Cursor, showCursor bool) {
	lv := tm.linesView
	ln := lv.Lines
	if cleared {
		tm.pending = nil
		tm.numScrollback = 0
	}
	var text [][]rune
	var markup []rich.Text
	addLine := func(rows []*vt.Line, cursorOff int) {
		var cells []vt.Cell
		for _, r := range rows {
			cells = append(cells, r.Cells...)
		}
		cursor := -1
		if showCursor {
			cursor = cursorOff
		}
		rs, mu := cm.lineMarkup(cells, cursorOff+1, cursor)
		text = append(text, rs)
		markup = append(markup, mu)
	}
	// complete lines of scrollback
	rows := append(tm.pending, scrolled...)
	st := 0
	for i, r := range rows {
		if !r.Wrapped {
			addLine(rows[st:i+1], -1)
			st = i + 1
		}
	}
	tm.pending = slices.Clone(rows[st:])
	nhist := len(text)

	// the screen, with the pending lines
	live := append(slices.Clone(tm.pending), screen...)
	crow := len(tm.pending) + cur.Row
	tm.linesCursor = textpos.Pos{Line: -1}
	st = 0
	for i, r := range live {
		if !r.Wrapped || i == len(live)-1 {
			off := -1
			if crow >= st && crow <= i {
				off = cur.Col
				for _, pr := range live[st:crow] {
					off += len(pr.Cells)
				}
				tm.linesCursor = textpos.Pos{Line: tm.numScrollback + len(text), Char: off}
			}
			addLine(live[st:i+1], off)
			st = i + 1
		}
	}
	ln.ReplaceLinesMarkup(tm.numScrollback, -1, text, markup)
	tm.numScrollback += nhist
	if over := tm.numScrollback - tm.MaxScrollback; tm.MaxScrollback > 0 && over > tm.MaxScrollback/10 {
		ln.ReplaceLinesMarkup(0, over, nil, nil)
		tm.numScrollback -= over
		tm.linesCursor.Line -= over
	}
}

// cursorColor returns the color of the cursor, and whether the view
// that shows it is focused, in which case the color is blended with
// the background of the cell.
func (tm *Terminal) cursorColor() (color.RGBA, bool) {
	vw := tm.ActiveView()
	if vw != nil && vw.StateIs(states.Focused) {
		return colors.ToUniform(colors.Scheme.Primary.Base), true
	}
	return colors.ToUniform(colors.Scheme.OutlineVariant), false
}

// sendInfo calls the listener functions for changes in the title,
// directory, and commands, and for the exit of the program.
func (tm *Terminal) sendInfo() {
	if !tm.infoChanged.Swap(false) {
		return
	}
	t := tm.emulator
	t.Lock()
	title, dir := t.Title, t.Dir
	var done []vt.Command
	for i := tm.numCommands; i < len(t.Commands); i++ {
		c := t.Commands[i]
		if !c.Done {
			break
		}
		done = append(done, *c)
		tm.numCommands = i + 1
	}
	if tm.numCommands > len(t.Commands) { // trimmed
		tm.numCommands = len(t.Commands)
	}
	t.Unlock()
	if title != tm.title {
		tm.title = title
		for _, f := range tm.titleFuncs {
			f(title)
		}
	}
	if dir != tm.dir {
		tm.dir = dir
		for _, f := range tm.dirFuncs {
			f(dir)
		}
	}
	for i := range done {
		for _, f := range tm.commandFuncs {
			f(&done[i])
		}
	}
	if tm.exited.Load() && !tm.exitSent {
		tm.exitSent = true
		for _, f := range tm.exitFuncs {
			f(tm.exitErr)
		}
	}
}
