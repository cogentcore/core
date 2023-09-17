// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
)

// Usage returns a usage string based on the given
// configuration struct and commands. It contains [AppAbout],
// a list of commands and their descriptions, and a list of
// flags and their descriptions. The resulting string uses
// color escape codes.
func Usage[T any](opts *Options, cfg T, cmd string, cmds ...*Cmd[T]) string {
	var b strings.Builder
	if cmd == "" {
		if opts.AppAbout != "" {
			b.WriteString(opts.AppAbout)
			b.WriteString("\n\n")
		}
	} else {
		gotCmd := false
		for _, c := range cmds {
			if c.Name == cmd {
				if c.Doc != "" {
					b.WriteString(CmdColor(cmd) + " " + c.Doc)
					b.WriteString("\n\n")
				}
				gotCmd = true
				break
			}
		}
		if !gotCmd {
			fmt.Println(CmdColor(opts.AppName+" help") + ErrorColor(" failed: command %q not found", cmd))
			os.Exit(1)
		}
	}

	fields := &Fields{}
	AddFields(cfg, fields, cmd)

	cmdName := opts.AppName
	if cmd != "" {
		cmdName += " " + cmd
	}
	b.WriteString("Usage:\n\t" + CmdColor(cmdName+" "))

	posArgStrs := []string{}

	for _, kv := range fields.Order {
		v := kv.Val
		f := v.Field

		posArgTag, ok := f.Tag.Lookup("posarg")
		if ok {
			ui, err := strconv.ParseUint(posArgTag, 10, 64)
			if err != nil {
				fmt.Printf(ErrorColor("programmer error:")+" invalid value %q for posarg struct tag on field %q: %v\n", posArgTag, f.Name, err)
			}
			// if the slice isn't big enough, grow it to fit this posarg
			if ui >= uint64(len(posArgStrs)) {
				posArgStrs = slices.Grow(posArgStrs, len(posArgStrs)-int(ui)+1) // increase capacity
				posArgStrs = posArgStrs[:ui+1]                                  // extend to capacity
			}
			nm := strcase.ToKebab(v.Names[0])
			req, has := f.Tag.Lookup("required")
			if req == "+" || req == "true" || !has { // default is required, so !has => required
				posArgStrs[ui] = "<" + nm + ">"
			} else {
				posArgStrs[ui] = "[" + nm + "]"
			}

		}
	}
	b.WriteString(CmdColor(strings.Join(posArgStrs, " ")))
	if len(posArgStrs) > 0 {
		b.WriteString(" ")
	}
	b.WriteString(CmdColor("[flags]\n"))

	CommandUsage(&b, cmdName, cmd, cmds...)

	b.WriteString("\nThe flags are: (flags are case-insensitive, can be in kebab-case,\n")
	b.WriteString("snake_case, or CamelCase, and can have one or two leading dashes)\n\n")

	b.WriteString(CmdColor("-help") + " or " + CmdColor("-h") + "\n\tshow usage information for a command\n")
	b.WriteString(CmdColor("-config") + " or " + CmdColor("-cfg") + "\n\tthe filename to load configuration options from\n")
	FlagUsage(fields, &b)
	return b.String()
}

// CommandUsage adds the command usage info for the given commands to the
// given [strings.Builder]. Typically, end-user code should use [Usage] instead.
// It also takes the full name of our command as it appears in the terminal (cmdName),
// (eg: "goki build"), and the name of the command we are running (eg: "build").
//
// To be a command that is included in the usage, we must be one command
// nesting depth (subcommand) deeper than the current command (ie, if we
// are on "x", we can see usage for commands of the form "x y"), and all
// of our commands must be consistent with the current command. For example,
// "" could generate usage for "help", "build", and "run", and "mod" could
// generate usage for "mod init", "mod tidy", and "mod edit". This ensures
// that only relevant commands are shown in the usage.
func CommandUsage[T any](b *strings.Builder, cmdName string, cmd string, cmds ...*Cmd[T]) {
	acmds := []*Cmd[T]{}           // actual commands we care about
	var rcmd *Cmd[T]               // root command
	cmdstrs := strings.Fields(cmd) // subcommand strings in passed command

	// need this label so that we can continue outer loop when we have non-matching cmdstr
outer:
	for _, c := range cmds {
		cstrs := strings.Fields(c.Name)   // subcommand strings in command we are checking
		if len(cstrs) != len(cmdstrs)+1 { // we must be one deeper
			continue
		}
		for i, cmdstr := range cmdstrs {
			if cmdstr != cstrs[i] { // every subcommand so far must match
				continue outer
			}
		}
		if c.Root {
			rcmd = c
		} else if c.Name != cmd { // if it is the same subcommand we are already on, we handle it above in main Usage
			acmds = append(acmds, c)
		}
	}

	if len(acmds) != 0 {
		b.WriteString("\t" + CmdColor(cmdName+" <subcommand> [flags]\n"))
	}

	if rcmd != nil {
		b.WriteString("\nThe default (root) command is:\n")
		b.WriteString("\t" + CmdColor(rcmd.Name) + "\t" + strings.ReplaceAll(rcmd.Doc, "\n", "\n\t") + "\n") // need to put a tab on every newline for formatting
	}

	if len(acmds) == 0 && cmd != "" { // nothing to do
		return
	}

	b.WriteString("\nThe subcommands are:\n")

	// if we are in root, we also add help
	if cmd == "" {
		b.WriteString("\t" + CmdColor("help") + "\tshows usage information for a command\n")
	}

	for _, c := range acmds {
		b.WriteString("\t" + CmdColor(c.Name))
		if c.Doc != "" {
			b.WriteString("\t" + strings.ReplaceAll(c.Doc, "\n", "\n\t")) // need to put a tab on every newline for formatting
		}
		b.WriteString("\n")
	}
}

// FlagUsage adds the flag usage info for the
// given fields to the given [strings.Builder].
// Typically, you should use [Usage] instead.
func FlagUsage(fields *Fields, b *strings.Builder) {
	for _, kv := range fields.Order {
		f := kv.Val
		for i, name := range f.Names {
			b.WriteString(CmdColor("-" + strcase.ToKebab(name)))
			// handle English sentence construction with "or" and commas
			if i == len(f.Names)-2 {
				if len(f.Names) > 2 {
					b.WriteString(",")
				}
				b.WriteString(" or ")
			} else if i != len(f.Names)-1 {
				b.WriteString(", ")
			}
		}
		b.WriteString("\n")
		desc, hast := f.Field.Tag.Lookup("desc")
		if hast && desc != "" {
			b.WriteString("\t" + strings.ReplaceAll(desc, "\n", "\n\t")) // need to put a tab on every newline for formatting
			def, ok := f.Field.Tag.Lookup("def")
			if ok && def != "" {
				b.WriteString(fmt.Sprintf(" (default: %s)", def))
			}
		}
		b.WriteString("\n")
	}
}
