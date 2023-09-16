// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"goki.dev/ki/v2/kit"
)

// Usage returns a usage string based on the given
// configuration struct and commands. It contains [AppAbout],
// a list of commands and their descriptions, and a list of
// flags and their descriptions. The resulting string uses
// color escape codes.
func Usage[T any](opts *Options, cfg T, cmd string, cmds ...*Cmd[T]) string {
	var b strings.Builder
	if cmd == "" {
		b.WriteString(opts.AppAbout)
		b.WriteString("\n\n")
	} else {
		gotCmd := false
		for _, c := range cmds {
			if c.Name == cmd {
				b.WriteString(c.Doc)
				b.WriteString("\n\n")
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
	b.WriteString("The following commands are available:\n\n")
	// for _, c := range cmds {
	// 	if c.Root {
	// 		b.WriteString(cmdColor("<default command>") + "\n\t" + c.Doc + "\n")
	// 		break
	// 	}
	// }
	b.WriteString(cmdColor("help") + "\n\tshow this usage message and exit\n")
	CommandUsage(&b, cmd, cmds...)
	b.WriteString("\n")

	b.WriteString("The following flags are available. Flags are case-insensitive and\n")
	b.WriteString("can be in kebab-case, snake_case, or CamelCase. Also, there can be\n")
	b.WriteString("one or two leading dashes. Most flags can be used without nesting\n")
	b.WriteString("paths (e.g. -target instead of -build-target)\n\n")

	b.WriteString(cmdColor("-help") + " or " + cmdColor("-h") + "\n\tshow this usage message and exit\n")
	b.WriteString(cmdColor("-config") + " or " + cmdColor("-cfg") + "\n\tthe filename to load configuration options from\n")
	FlagUsage(cfg, "", &b, cmd)
	return b.String()
}

// CommandUsage adds the command usage info for
// the given commands to the given [strings.Builder].
// Typically, you should use [Usage] instead.
func CommandUsage[T any](b *strings.Builder, cmd string, cmds ...*Cmd[T]) {
	for _, c := range cmds {
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
}

// FlagUsage adds the flag usage info for the
// given app to the given [strings.Builder].
// Typically, you should use [Usage] instead.
// Pass an empty string for path unless you are
// already in a nested context, which should only
// happen internally (if you don't know whether
// you're in a nested context, you're not).
func FlagUsage(app any, path string, b *strings.Builder, cmd string) {
	typ := kit.NonPtrType(reflect.TypeOf(app))
	val := kit.NonPtrValue(reflect.ValueOf(app))
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fv := val.Field(i)
		cmdtag, ok := f.Tag.Lookup("cmd")
		if ok && cmdtag != cmd { // if we are associated with a different command, skip
			continue
		}
		if kit.NonPtrType(f.Type).Kind() == reflect.Struct {
			nwPath := f.Name
			if path != "" {
				nwPath = path + "." + nwPath
			}
			FlagUsage(kit.PtrValue(fv).Interface(), nwPath, b, cmd)
			continue
		}
		nm := f.Name
		if nm == "Includes" {
			continue
		}
		if path != "" {
			nm = path + "." + nm
		}
		b.WriteString(cmdColor("-" + strcase.ToKebab(nm) + "\n"))
		desc, ok := f.Tag.Lookup("desc")
		if ok && desc != "" {
			b.WriteString("\t" + strings.ReplaceAll(desc, "\n", "\n\t")) // need to put a tab on every newline for formatting
			def, ok := f.Tag.Lookup("def")
			if ok && def != "" {
				b.WriteString(fmt.Sprintf(" (default: %s)", def))
			}
		}
		b.WriteString("\n")
	}
}
