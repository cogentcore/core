// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vt

import (
	"encoding/base64"
	"fmt"
	"image/color"
	"net/url"
	"strconv"
	"strings"
)

// handleOSC handles an operating system command, which has the
// command number and then its data: "cmd;data".
func (em *Emulator) handleOSC(cmd int, data []byte) {
	s := string(data)
	if i := strings.IndexByte(s, ';'); i >= 0 {
		s = s[i+1:]
	} else {
		s = ""
	}
	switch cmd {
	case 0, 2:
		em.Title = s
		em.changed()
	case 7:
		em.setDir(dirFromURL(s))
	case 8:
		em.setLink(s)
	case 10:
		if s == "?" {
			em.reply(colorReply(10, em.Foreground))
		}
	case 11:
		if s == "?" {
			em.reply(colorReply(11, em.Background))
		}
	case 52:
		em.setClipboard(s)
	case 133:
		em.shellMark(s, false)
	case 633:
		em.shellMark(s, true)
	case 1337:
		if dir, ok := strings.CutPrefix(s, "CurrentDir="); ok {
			em.setDir(dir)
		}
	}
}

// setDir sets the current working directory.
func (em *Emulator) setDir(dir string) {
	if dir == "" || dir == em.Dir {
		return
	}
	em.Dir = dir
	if len(em.Commands) > 0 {
		if c := em.Commands[len(em.Commands)-1]; !c.Done && c.Dir == "" {
			c.Dir = dir
		}
	}
	em.changed()
}

// dirFromURL returns the path of a file URL, as sent with OSC 7.
func dirFromURL(s string) string {
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "file" {
		return ""
	}
	p := u.Path
	if len(p) > 2 && p[0] == '/' && p[2] == ':' { // Windows drive: /C:/dir
		p = p[1:]
	}
	return p
}

// colorReply returns the reply to a query for the given color.
func colorReply(cmd int, c color.RGBA) string {
	return fmt.Sprintf("\x1b]%d;rgb:%04x/%04x/%04x\x1b\\", cmd, uint16(c.R)*0x101, uint16(c.G)*0x101, uint16(c.B)*0x101)
}

// setLink starts or ends a hyperlink (OSC 8): "params;uri",
// with an empty uri ending it.
func (em *Emulator) setLink(s string) {
	_, uri, _ := strings.Cut(s, ";")
	if uri == "" {
		em.link = 0
		return
	}
	for i, l := range em.Links {
		if l == uri {
			em.link = uint32(i + 1)
			return
		}
	}
	em.Links = append(em.Links, uri)
	em.link = uint32(len(em.Links))
}

// setClipboard handles OSC 52: "selection;base64 data".
func (em *Emulator) setClipboard(s string) {
	if em.Clipboard == nil {
		return
	}
	_, d, _ := strings.Cut(s, ";")
	if d == "?" { // queries are not answered, for security
		return
	}
	b, err := base64.StdEncoding.DecodeString(d)
	if err != nil {
		return
	}
	em.Clipboard(string(b))
}

// Command is a command run in the shell, as reported by the shell with
// shell integration sequences: OSC 133 (FinalTerm) or OSC 633 (VS Code).
// Lines are absolute line numbers (see [Emulator.NumScrolled]).
type Command struct {
	// PromptLine is the line where the prompt starts.
	PromptLine int

	// InputLine and InputCol are where the command input starts,
	// after the prompt.
	InputLine, InputCol int

	// OutputLine is the line where the output of the command starts,
	// or -1 before it is run.
	OutputLine int

	// EndLine is the line where the command finished, or -1.
	EndLine int

	// Text is the command line.
	Text string

	// Dir is the working directory of the command, if known.
	Dir string

	// ExitCode is the exit code of the command, if it is Done
	// and the shell reported it, or -1.
	ExitCode int

	// Done is whether the command has finished.
	Done bool
}

// shellMark handles a shell integration mark: A prompt start,
// B command start, C command executed (output start), D[;exit code]
// command finished, and for OSC 633 (vscode), E;command line and
// P;Cwd=dir.
func (em *Emulator) shellMark(s string, vscode bool) {
	if s == "" {
		return
	}
	kind, arg, _ := strings.Cut(s, ";")
	line := em.numScrolled + em.Cursor.Row
	last := func() *Command {
		if len(em.Commands) == 0 {
			return nil
		}
		return em.Commands[len(em.Commands)-1]
	}
	switch kind {
	case "A":
		if c := last(); c != nil && !c.Done && c.OutputLine < 0 {
			c.PromptLine = line // a new prompt without running a command
			return
		}
		em.Commands = append(em.Commands, &Command{PromptLine: line, InputLine: -1, OutputLine: -1, EndLine: -1, ExitCode: -1, Dir: em.Dir})
		if len(em.Commands) > em.MaxCommands && em.MaxCommands > 0 {
			em.Commands = em.Commands[len(em.Commands)-em.MaxCommands:]
		}
	case "B":
		if c := last(); c != nil && !c.Done {
			c.InputLine, c.InputCol = line, em.Cursor.Col
		}
	case "C":
		if c := last(); c != nil && !c.Done {
			c.OutputLine = line
			if c.Text == "" {
				c.Text = em.inputText(c)
			}
		}
	case "D":
		if c := last(); c != nil && !c.Done {
			c.Done = true
			c.EndLine = line
			if code, err := strconv.Atoi(strings.TrimSpace(arg)); err == nil {
				c.ExitCode = code
			}
		}
	case "E":
		if c := last(); c != nil && vscode {
			cmd, _, _ := strings.Cut(arg, ";")
			c.Text = unescapeVSCode(cmd)
		}
		return
	case "P":
		if dir, ok := strings.CutPrefix(arg, "Cwd="); ok {
			em.setDir(unescapeVSCode(dir))
		}
		return
	default:
		return
	}
	em.changed()
}

// inputText returns the text of the command input from the screen,
// from its start up to the cursor line, if it is still on the screen.
func (em *Emulator) inputText(c *Command) string {
	if c.InputLine < 0 {
		return ""
	}
	st := c.InputLine - em.numScrolled
	ed := em.Cursor.Row
	if em.Cursor.Col == 0 && ed > st {
		ed--
	}
	if st < 0 || st > ed || ed >= em.Rows {
		return ""
	}
	var sb strings.Builder
	for r := st; r <= ed; r++ {
		ln := em.screen.lines[r]
		rs := ln.Runes(len(ln.Cells))
		if r == st {
			rs = rs[min(c.InputCol, len(rs)):]
		}
		if !ln.Wrapped {
			rs = []rune(strings.TrimRight(string(rs), " "))
		}
		sb.WriteString(string(rs))
		if !ln.Wrapped && r < ed {
			sb.WriteByte('\n')
		}
	}
	return strings.TrimSpace(sb.String())
}

// unescapeVSCode unescapes the \xAB hex escapes in OSC 633 values.
func unescapeVSCode(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			if s[i+1] == '\\' {
				sb.WriteByte('\\')
				i++
				continue
			}
			if s[i+1] == 'x' && i+3 < len(s) {
				if b, err := strconv.ParseUint(s[i+2:i+4], 16, 8); err == nil {
					sb.WriteByte(byte(b))
					i += 3
					continue
				}
			}
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}
