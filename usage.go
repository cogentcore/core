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

// Indent is the value used for indentation in [Usage].
var Indent = "    "

// Usage returns a usage string based on the given options,
// configuration struct, current command, and available commands.
// It contains [AppAbout], a list of commands and their descriptions,
// and a list of flags and their descriptions, scoped based on the
// current command and its associated commands and configuration.
// The resulting string contains color escape codes.
func Usage[T any](opts *Options, cfg T, cmd string, cmds ...*Cmd[T]) string {
	var b strings.Builder
	if cmd == "" {
		if opts.AppAbout != "" {
			b.WriteString("\n" + opts.AppAbout + "\n\n")
		}
	} else {
		gotCmd := false
		for _, c := range cmds {
			if c.Name == cmd {
				if c.Doc != "" {
					b.WriteString("\n" + c.Doc + "\n\n")
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
	b.WriteString(HeaderColor("Usage:\n") + Indent + CmdColor(cmdName+" "))

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
				posArgStrs[ui] = CmdColor("<" + nm + ">")
			} else {
				posArgStrs[ui] = SuccessColor("[" + nm + "]")
			}

		}
	}
	b.WriteString(strings.Join(posArgStrs, " "))
	if len(posArgStrs) > 0 {
		b.WriteString(" ")
	}
	b.WriteString(SuccessColor("[flags]\n"))

	CommandUsage(&b, cmdName, cmd, cmds...)

	b.WriteString(HeaderColor("\nFlags:\n") + Indent + InfoColor("Flags are case-insensitive, can be in kebab-case, snake_case,\n"))
	b.WriteString(Indent + InfoColor("or CamelCase, and can have one or two leading dashes.\n\n"))

	b.WriteString(Indent + CmdColor("-help") + ", " + CmdColor("-h") + SuccessColor(" bool") + "\n" + Indent + Indent + "show usage information for a command\n")
	b.WriteString(Indent + CmdColor("-config") + ", " + CmdColor("-cfg") + SuccessColor(" filename") + "\n" + Indent + Indent + "the filename to load configuration options from\n")
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
		b.WriteString(Indent + CmdColor(cmdName+" <subcommand> ") + SuccessColor("[flags]\n"))
	}

	if rcmd != nil {
		b.WriteString(HeaderColor("\nDefault command:\n"))
		b.WriteString(Indent + CmdColor(rcmd.Name) + "\n" + Indent + Indent + strings.ReplaceAll(rcmd.Doc, "\n", "\n"+Indent+Indent) + "\n") // need to put two indents on every newline for formatting
	}

	if len(acmds) == 0 && cmd != "" { // nothing to do
		return
	}

	b.WriteString(HeaderColor("\nSubcommands:\n"))

	// if we are in root, we also add help
	if cmd == "" {
		b.WriteString(Indent + CmdColor("help") + "\n" + Indent + Indent + "help shows usage information for a command\n")
	}

	for _, c := range acmds {
		b.WriteString(Indent + CmdColor(c.Name))
		if c.Doc != "" {
			b.WriteString("\n" + Indent + Indent + strings.ReplaceAll(c.Doc, "\n", "\n"+Indent+Indent)) // need to put two indents on every newline for formatting
		}
		b.WriteString("\n")
	}
}

// FlagUsage adds the flag usage info for the given fields
// to the given [strings.Builder]. Typically, end-user code
// should use [Usage] instead.
func FlagUsage(fields *Fields, b *strings.Builder) {
	for _, kv := range fields.Order {
		f := kv.Val
		b.WriteString(Indent)
		for i, name := range f.Names {
			b.WriteString(CmdColor("-" + strcase.ToKebab(name)))
			if i != len(f.Names)-1 {
				b.WriteString(", ")
			}
		}
		b.WriteString(" " + SuccessColor(f.Field.Type.String()))
		b.WriteString("\n")
		desc, hast := f.Field.Tag.Lookup("desc")
		if hast && desc != "" {
			b.WriteString(Indent + Indent + strings.ReplaceAll(desc, "\n", "\n"+Indent+Indent)) // need to put two indents on every newline for formatting
			def, ok := f.Field.Tag.Lookup("def")
			if ok && def != "" {
				b.WriteString(fmt.Sprintf(" (default: %s)", def))
			}
		}
		b.WriteString("\n")
	}
}
