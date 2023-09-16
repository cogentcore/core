// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"goki.dev/laser"
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
					b.WriteString(c.Doc)
					b.WriteString("\n\n")
				}
				gotCmd = true
				break
			}
		}
		if !gotCmd {
			fmt.Println(cmdColor(opts.AppName+" help") + errorColor(" failed: command %s not found", cmd))
			os.Exit(1)
		}
	}
	cmdName := opts.AppName
	if cmd != "" {
		cmdName += " " + cmd
	}
	b.WriteString("Usage: " + cmdColor(cmdName+" [command] [arguments] [flags]\n\n"))
	CommandUsage(&b, cmd, cmds...)

	b.WriteString("The following flags are available. Flags are case-insensitive and\n")
	b.WriteString("can be in kebab-case, snake_case, or CamelCase. Also, there can be\n")
	b.WriteString("one or two leading dashes. Most flags can be used without nesting\n")
	b.WriteString("paths (e.g. -target instead of -build-target)\n\n")

	b.WriteString(cmdColor("-help") + " or " + cmdColor("-h") + "\n\tshow usage information for a command\n")
	b.WriteString(cmdColor("-config") + " or " + cmdColor("-cfg") + "\n\tthe filename to load configuration options from\n")
	FlagUsage(cfg, "", &b, cmd)
	return b.String()
}

// CommandUsage adds the command usage info for the given commands to the
// given [strings.Builder]. Typically, end-user code should use [Usage] instead.
//
// To be a command that is included in the usage, we must be one command
// nesting depth (subcommand) deeper than the current command (ie, if we
// are on "x", we can see usage for commands of the form "x y"), and all
// of our commands must be consistent with the current command. For example,
// "" could generate usage for "help", "build", and "run", and "mod" could
// generate usage for "mod init", "mod tidy", and "mod edit". This ensures
// that only relevant commands are shown in the usage.
func CommandUsage[T any](b *strings.Builder, cmd string, cmds ...*Cmd[T]) {
	b.WriteString("The following commands are available:\n\n")
	for _, c := range cmds {
		if (c.Root && cmd == "") || c.Name == cmd {
			b.WriteString(cmdColor("<default command> ("+c.Name+")") + "\n\t" + strings.ReplaceAll(c.Doc, "\n", "\n\t") + "\n") // need to put a tab on every newline for formatting
			break
		}
	}
	// if we are in root, we also add help
	if cmd == "" {
		b.WriteString(cmdColor("help") + "\n\tshow usage information for a command\n")
	}

	cmdstrs := strings.Fields(cmd)
	// need this label so that we can continue outer loop when we have non-matching cmdstr
outer:
	for _, c := range cmds {
		if (c.Root && cmd == "") || c.Name == cmd { // we already handled this case above, so skip
			continue
		}
		cstrs := strings.Fields(c.Name)
		if len(cstrs) != len(cmdstrs)+1 {
			continue
		}
		for i, cmdstr := range cmdstrs {
			if cmdstr != cstrs[i] {
				continue outer
			}
		}
		if c.Root {
			b.WriteString(cmdColor(c.Name + " (default command)"))
		} else {
			b.WriteString(cmdColor(c.Name))
		}
		if c.Doc != "" {
			b.WriteString("\n\t" + strings.ReplaceAll(c.Doc, "\n", "\n\t")) // need to put a tab on every newline for formatting
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
}

// FlagUsage adds the flag usage info for the
// given app to the given [strings.Builder].
// Typically, you should use [Usage] instead.
// Pass an empty string for path unless you are
// already in a nested context, which should only
// happen internally (if you don't know whether
// you're in a nested context, you're not).
func FlagUsage(app any, path string, b *strings.Builder, cmd string) {
	typ := laser.NonPtrType(reflect.TypeOf(app))
	val := laser.NonPtrValue(reflect.ValueOf(app))
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fv := val.Field(i)
		cmdtag, hast := f.Tag.Lookup("cmd")
		if hast && cmdtag != cmd { // if we are associated with a different command, skip
			continue
		}
		if laser.NonPtrType(f.Type).Kind() == reflect.Struct {
			nwPath := f.Name
			// if we are scoped by command, we don't need new path scope
			// TODO: need a better approach to this; maybe use allFlags map
			if hast {
				nwPath = ""
			}
			if path != "" {
				nwPath = path + "." + nwPath
			}
			FlagUsage(laser.PtrValue(fv).Interface(), nwPath, b, cmd)
			continue
		}
		if f.Name == "Includes" {
			continue
		}
		names := []string{f.Name}
		greasetag, hast := f.Tag.Lookup("grease")
		if hast {
			names = strings.Split(greasetag, ",")
			if len(names) == 0 {
				log.Fatalln("expected at least one name in grease struct tag, but got none")
			}
		}

		if path != "" {
			for i, name := range names {
				names[i] = path + "." + name
			}
		}
		for i, name := range names {
			b.WriteString(cmdColor("-" + strcase.ToKebab(name)))
			// handle English sentence construction with "or" and commas
			if i == len(names)-2 {
				if len(names) > 2 {
					b.WriteString(",")
				}
				b.WriteString(" or ")
			} else if i != len(names)-1 {
				b.WriteString(", ")
			}
		}
		b.WriteString("\n")
		desc, hast := f.Tag.Lookup("desc")
		if hast && desc != "" {
			b.WriteString("\t" + strings.ReplaceAll(desc, "\n", "\n\t")) // need to put a tab on every newline for formatting
			def, ok := f.Tag.Lookup("def")
			if ok && def != "" {
				b.WriteString(fmt.Sprintf(" (default: %s)", def))
			}
		}
		b.WriteString("\n")
	}
}
