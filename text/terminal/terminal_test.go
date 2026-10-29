// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/terminal/vt"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

// testShell is the shell of the test terminals, which is dash on Debian
// and Ubuntu, so the tests must not depend on line editing.
const testShell = "/bin/sh"

// lastTerminal is the last test terminal, whose text is logged when
// [waitFor] times out.
var lastTerminal *Terminal

// testTerminal runs a terminal with a POSIX shell with a simple prompt
// in a new window, which is closed at the end of the test.
func testTerminal(t *testing.T) (*core.Body, *Terminal) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	b := core.NewBody()
	tm := NewTerminal(b).SetCommand(testShell).SetEnv("PS1=$ ", "ENV=", "HISTFILE=")
	lastTerminal = tm
	tm.Styler(func(s *styles.Style) {
		s.Min.Set(units.Ch(40), units.Em(10))
	})
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) { showed <- struct{}{} })
	b.RunWindow()
	<-showed
	t.Cleanup(func() { // the window is closed, which stops the program
		b.AsyncLock()
		b.Close()
		b.AsyncUnlock()
	})
	waitFor(t, b, "the prompt", func() bool {
		return strings.Contains(tm.linesView.Lines.String(), "$ ")
	})
	return b, tm
}

// waitFor waits until the given condition, which is checked
// with the window locked, is true.
func waitFor(t *testing.T, b *core.Body, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		b.AsyncLock()
		ok := cond()
		b.AsyncUnlock()
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	b.AsyncLock()
	defer b.AsyncUnlock()
	if lastTerminal != nil {
		t.Logf("terminal text: %q", linesText(lastTerminal))
	}
	t.Fatalf("timeout waiting for %s", what)
}

// hasOutputLine returns whether the lines view has a line of output
// with the given text, which can follow a prompt: shells without line
// editing (such as dash, the sh of Debian and Ubuntu) echo input that is
// typed ahead before the prompt for it, so output follows a prompt.
func hasOutputLine(tm *Terminal, s string) bool {
	for _, ln := range strings.Split(linesText(tm), "\n") {
		if strings.TrimPrefix(ln, "$ ") == s {
			return true
		}
	}
	return false
}

// linesText returns the text of the lines view, without trailing blank lines.
func linesText(tm *Terminal) string {
	return strings.TrimRight(tm.linesView.Lines.String(), " \n")
}

func TestTerminalRun(t *testing.T) {
	b, tm := testTerminal(t)
	b.AsyncLock()
	assert.Greater(t, tm.size.X, 10)
	assert.Greater(t, tm.size.Y, 5)
	tm.Run("echo hello")
	b.AsyncUnlock()
	waitFor(t, b, "the output", func() bool {
		return strings.HasSuffix(linesText(tm), "$ echo hello\nhello\n$")
	})
	b.AsyncLock()
	defer b.AsyncUnlock()
	assert.False(t, tm.altShown)
	cur := tm.linesCursor
	assert.Equal(t, 2, cur.Char, "the cursor is after the prompt")
	assert.Equal(t, "$ ", string(tm.linesView.Lines.Line(cur.Line)[:2]))
}

func TestTerminalScrollback(t *testing.T) {
	b, tm := testTerminal(t)
	b.AsyncLock()
	tm.Run("i=1; while [ $i -le 300 ]; do echo line $i; i=$((i+1)); done")
	b.AsyncUnlock()
	waitFor(t, b, "the output", func() bool {
		return strings.HasSuffix(linesText(tm), "line 300\n$")
	})
	b.AsyncLock()
	defer b.AsyncUnlock()
	txt := linesText(tm)
	for _, ln := range []string{"line 1\n", "line 150\n", "line 299\n"} {
		assert.Contains(t, txt, ln)
	}
	assert.Greater(t, tm.numScrollback, 250, "lines in the scrollback")

	tm.ClearScrollback()
	assert.Equal(t, "$", linesText(tm))
	assert.Equal(t, 0, tm.numScrollback)
}

func TestTerminalWrap(t *testing.T) {
	b, tm := testTerminal(t)
	b.AsyncLock()
	cols := tm.size.X
	long := strings.Repeat("abcdefghi ", cols/10*3) // three lines worth
	tm.Run("echo " + long)
	b.AsyncUnlock()
	waitFor(t, b, "the output", func() bool {
		return strings.Contains(linesText(tm), "\n"+strings.TrimSpace(long)+"\n$")
	})
}

func TestTerminalAltScreen(t *testing.T) {
	b, tm := testTerminal(t)
	b.AsyncLock()
	tm.Run(`printf '\033[?1049h\033[2;3Hfull screen'; read x; printf '\033[?1049l'`)
	b.AsyncUnlock()
	waitFor(t, b, "the alternate screen", func() bool {
		return tm.altShown && strings.Contains(tm.gridView.Lines.String(), "full screen")
	})
	b.AsyncLock()
	assert.Equal(t, 1, tm.StackTop)
	assert.Equal(t, "\n  full screen", strings.TrimRight(tm.gridView.Lines.String(), "\n "))
	assert.NotContains(t, linesText(tm), "\n  full screen")
	tm.Run("")
	b.AsyncUnlock()
	waitFor(t, b, "the primary screen", func() bool {
		return !tm.altShown && strings.HasSuffix(linesText(tm), "\n$")
	})
	b.AsyncLock()
	assert.Equal(t, 0, tm.StackTop)
	b.AsyncUnlock()
}

func TestTerminalCd(t *testing.T) {
	b, tm := testTerminal(t)
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	sub := filepath.Join(dir, "it's a dir")
	require.NoError(t, os.Mkdir(sub, 0o755))
	b.AsyncLock()
	tm.Cd(sub)
	tm.Run("pwd")
	b.AsyncUnlock()
	waitFor(t, b, "the directory", func() bool {
		return hasOutputLine(tm, sub)
	})
}

func TestTerminalShellIntegration(t *testing.T) {
	b, tm := testTerminal(t)
	var done []string
	var dirs []string
	b.AsyncLock()
	tm.OnCommand(func(c *vt.Command) { done = append(done, c.Text) })
	tm.OnDir(func(dir string) { dirs = append(dirs, dir) })
	// what shell integration scripts send
	tm.Run(`printf '\033]7;file://host/tmp/x\007\033]133;A\007$ \033]133;B\007ls\r\n\033]133;C\007out\r\n\033]133;D;0\007'`)
	b.AsyncUnlock()
	waitFor(t, b, "the command", func() bool {
		return len(done) == 1
	})
	b.AsyncLock()
	defer b.AsyncUnlock()
	assert.Equal(t, []string{"ls"}, done)
	assert.Equal(t, []string{"/tmp/x"}, dirs)
	assert.Equal(t, "/tmp/x", tm.WorkingDir())
	cs := tm.Commands()
	require.Equal(t, 1, len(cs))
	assert.Equal(t, 0, cs[0].ExitCode)
}

func TestTerminalExit(t *testing.T) {
	b, tm := testTerminal(t)
	exited := false
	b.AsyncLock()
	tm.OnExit(func(err error) { exited = true })
	tm.Run("exit 3")
	b.AsyncUnlock()
	waitFor(t, b, "the exit", func() bool {
		return exited && strings.Contains(linesText(tm), "[process exited: exit status 3]")
	})
	b.AsyncLock()
	assert.False(t, tm.IsRunning())
	assert.True(t, tm.restartOnKey())
	b.AsyncUnlock()
	waitFor(t, b, "the restart", func() bool {
		return tm.IsRunning() && strings.HasSuffix(linesText(tm), "restart)\n$")
	})
}

func TestTerminalPaste(t *testing.T) {
	b, tm := testTerminal(t)
	b.AsyncLock()
	tm.Paste("echo one\necho two\n")
	b.AsyncUnlock()
	waitFor(t, b, "the output", func() bool {
		return hasOutputLine(tm, "one") && hasOutputLine(tm, "two")
	})
}

// renderTerminal renders a terminal that runs the given shell commands.
func renderTerminal(t *testing.T, name, cmds string, ready func(tm *Terminal) bool) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	b := core.NewBody()
	tm := NewTerminal(b).SetCommand("/bin/sh", "-c", cmds+"; read x")
	lastTerminal = tm
	tm.Styler(func(s *styles.Style) {
		s.Min.Set(units.Ch(50), units.Em(12))
	})
	b.AssertRender(t, name, func() {
		waitFor(t, b, "the output", func() bool { return ready(tm) })
	})
}

func TestRenderColors(t *testing.T) {
	cmds := `printf 'plain \033[1mbold\033[0m \033[3mitalic\033[0m \033[4munder\033[0m \033[2mfaint\033[0m \033[7mreverse\033[0m\r\n'
for i in 0 1 2 3 4 5 6 7; do printf "\033[3${i}m color$i \033[0m"; done; printf '\r\n'
for i in 0 1 2 3 4 5 6 7; do printf "\033[4${i}m bg$i \033[0m"; done; printf '\r\n'
printf '\033[38;2;255;128;0mtrue color\033[0m \033[48;5;22m 256 \033[0m wide: 世界! é\r\n'
printf 'a long line that wraps: the quick brown fox jumps over the lazy dog many times over'`
	renderTerminal(t, "colors", cmds, func(tm *Terminal) bool {
		return strings.Contains(linesText(tm), "times over")
	})
}

func TestRenderAltScreen(t *testing.T) {
	cmds := `printf 'below\r\n\033[?1049h\033[44m\033[2J\033[H\033(0lqqqqqqqqqqk\r\nx\033(B full \033(0x\r\nmqqqqqqqqqqj\033(B\033[0m\033[5;3H\033[1;33mbold yellow\033[0m'`
	renderTerminal(t, "alt-screen", cmds, func(tm *Terminal) bool {
		return tm.altShown && strings.Contains(tm.gridView.Lines.String(), "bold yellow")
	})
}

func TestTerminalKeys(t *testing.T) {
	b, tm := testTerminal(t)
	b.AsyncLock()
	tm.SetFocus()
	b.AsyncUnlock()
	src := b.Scene.RenderWindow().SystemWindow.Events()
	typeText := func(s string) {
		for _, r := range s {
			src.KeyChord(r, key.CodeUnknown, 0)
		}
	}
	press := func(code key.Codes, mods key.Modifiers) {
		src.Key(events.KeyDown, key.CodeRuneMap[code], code, mods)
		src.Key(events.KeyUp, key.CodeRuneMap[code], code, mods)
	}
	typeText("echo hellx")
	press(key.CodeBackspace, 0)
	typeText("o")
	press(key.CodeReturnEnter, 0)
	waitFor(t, b, "the output", func() bool {
		return strings.HasSuffix(linesText(tm), "$ echo hello\nhello\n$")
	})

	// Control+C interrupts the command
	typeText("sleep 20")
	press(key.CodeReturnEnter, 0)
	waitFor(t, b, "the command", func() bool {
		return strings.HasSuffix(linesText(tm), "$ sleep 20")
	})
	time.Sleep(100 * time.Millisecond)
	press(key.CodeC, mods(key.Control))
	waitFor(t, b, "the interrupt", func() bool {
		return strings.HasSuffix(linesText(tm), "\n$")
	})
}

func TestTerminalContextMenu(t *testing.T) {
	b, tm := testTerminal(t)
	b.AsyncLock()
	defer b.AsyncUnlock()
	tm.AddContextMenu(func(m *core.Scene, pos image.Point) {
		core.NewButton(m).SetText("App item")
	})
	m := core.NewScene()
	cms := tm.linesView.ContextMenus
	for i := len(cms) - 1; i >= 0; i-- {
		cms[i](m, image.Point{})
	}
	var items []string
	m.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		switch w := cw.(type) {
		case *core.Button:
			items = append(items, w.Text)
		case *core.Separator:
			items = append(items, "-")
		}
		return tree.Continue
	})
	assert.Equal(t, []string{"App item", "-", "Copy", "Paste", "Clear scrollback"}, items)
}

// TestTerminalKillKey tests that the key of [keymap.Kill] in a terminal
// (see [ShortcutChord]) clears the scrollback.
func TestTerminalKillKey(t *testing.T) {
	b, tm := testTerminal(t)
	b.AsyncLock()
	tm.Run("echo hello")
	tm.SetFocus()
	ch := ShortcutChord(keymap.Kill)
	b.AsyncUnlock()
	waitFor(t, b, "the output", func() bool {
		return strings.HasSuffix(linesText(tm), "hello\n$") && tm.linesView.StateIs(states.Focused)
	})
	require.NotEmpty(t, ch)
	rn, code, mods, err := ch.Chords()[0].Decode()
	require.NoError(t, err)
	src := b.Scene.RenderWindow().SystemWindow.Events()
	src.Key(events.KeyDown, rn, code, mods)
	waitFor(t, b, "the scrollback to be cleared", func() bool {
		return linesText(tm) == "$"
	})
}

// TestTerminalFunc tests a program that is a function of this process,
// with interrupts and size changes, which it gets as callbacks.
func TestTerminalFunc(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported")
	}
	b := core.NewBody()
	tm := NewTerminal(b)
	lastTerminal = tm
	tm.Styler(func(s *styles.Style) {
		s.Min.Set(units.Ch(40), units.Em(10))
	})
	tm.Func = func(tty *os.File) error {
		fd := int(tty.Fd())
		fmt.Fprintln(tty, "ready")
		rd := bufio.NewReader(tty)
		for {
			ln, err := rd.ReadString('\n')
			if err != nil {
				return err
			}
			switch ln = strings.TrimSpace(ln); ln {
			case "quit":
				return errors.New("quit")
			case "raw":
				st, err := term.MakeRaw(fd)
				if err != nil {
					return err
				}
				fmt.Fprint(tty, "raw\r\n")
				c, _ := rd.ReadByte()
				term.Restore(fd, st)
				fmt.Fprintf(tty, "byte %d\n", c)
			case "size":
				w, h, _ := term.GetSize(fd)
				fmt.Fprintf(tty, "size %d %d\n", w, h)
			default:
				fmt.Fprintf(tty, "got %s\n", ln)
			}
		}
	}
	interrupts := 0
	tm.OnInterrupt(func() { interrupts++ })
	var resized image.Point
	tm.OnResize(func(cols, rows int) { resized = image.Pt(cols, rows) })
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) { showed <- struct{}{} })
	b.RunWindow()
	<-showed
	defer func() {
		b.AsyncLock()
		b.Close()
		b.AsyncUnlock()
	}()
	waitFor(t, b, "the start", func() bool { return hasOutputLine(tm, "ready") })

	b.AsyncLock()
	tm.Run("hello")
	b.AsyncUnlock()
	waitFor(t, b, "the line", func() bool { return hasOutputLine(tm, "got hello") })

	// Control+C is an interrupt, unless the program reads raw input
	b.AsyncLock()
	tm.SendText("\x03")
	assert.Equal(t, 1, interrupts)
	tm.Run("raw")
	b.AsyncUnlock()
	waitFor(t, b, "raw input", func() bool { return hasOutputLine(tm, "raw") })
	b.AsyncLock()
	tm.SendText("\x03")
	assert.Equal(t, 1, interrupts)
	b.AsyncUnlock()
	waitFor(t, b, "the raw byte", func() bool { return hasOutputLine(tm, "byte 3") })

	b.AsyncLock()
	tm.viewSized(image.Pt(50, 12))
	assert.Equal(t, image.Pt(50, 12), resized)
	tm.Run("size")
	b.AsyncUnlock()
	waitFor(t, b, "the size", func() bool { return hasOutputLine(tm, "size 50 12") })

	b.AsyncLock()
	tm.Run("quit")
	b.AsyncUnlock()
	waitFor(t, b, "the exit", func() bool {
		return !tm.IsRunning() && strings.Contains(linesText(tm), "[process exited: quit]")
	})
	b.AsyncLock()
	assert.True(t, tm.restartOnKey())
	b.AsyncUnlock()
	waitFor(t, b, "the restart", func() bool {
		return tm.IsRunning() && strings.HasSuffix(linesText(tm), "restart)\nready")
	})
}
