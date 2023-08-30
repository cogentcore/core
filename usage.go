// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/goki/ki/kit"
	"github.com/iancoleman/strcase"
)

// Usage returns the usage string for the given app.
// It contains [AppAbout], a list of commands and
// their descriptions, and a list of flags and their
// descriptions.
func Usage(app any) string {
	var b strings.Builder
	b.WriteString(AppAbout)
	b.WriteString("\n\n")
	b.WriteString("The following commands are available:\n\n")

	b.WriteString("help\n\tshow this usage message and exit\n")
	CommandUsage(app, &b)
	b.WriteString("\n")

	b.WriteString("The following flags are available. Flags are case-insensitive and\n")
	b.WriteString("can be in kebab-case, snake_case, or CamelCase. Also, there can be\n")
	b.WriteString("one or two leading dashes. Most flags can be used without nesting\n")
	b.WriteString("paths (e.g. -target instead of -build-target)\n\n")

	b.WriteString("-help or -h\n\tshow this usage message and exit\n")
	b.WriteString("-config or -cfg\n\tthe filename to load configuration options from\n")
	FlagUsage(app, "", &b)
	return b.String()
}

// CommandUsage adds the command usage info for
// the given app to the given [strings.Builder].
// Typically, you should use [Usage] instead.
func CommandUsage(app any, b *strings.Builder) {
	typ := reflect.TypeOf(app)
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		if strings.HasSuffix(m.Name, "Cmd") {
			cmd := strcase.ToKebab(strings.TrimSuffix(m.Name, "Cmd"))
			b.WriteString(cmd)
			b.WriteString("\n")
		}
	}
}

// FlagUsage adds the flag usage info for the
// given app to the given [strings.Builder].
// Typically, you should use [Usage] instead.
// Pass an empty string for path unless you are
// already in a nested context, which should only
// happen internally (if you don't know whether
// you're in a nested context, you're not).
func FlagUsage(app any, path string, b *strings.Builder) {
	typ := kit.NonPtrType(reflect.TypeOf(app))
	val := kit.NonPtrValue(reflect.ValueOf(app))
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fv := val.Field(i)
		if kit.NonPtrType(f.Type).Kind() == reflect.Struct {
			nwPath := f.Name
			if path != "" {
				nwPath = path + "." + nwPath
			}
			FlagUsage(kit.PtrValue(fv).Interface(), nwPath, b)
			continue
		}
		nm := f.Name
		if nm == "Includes" {
			continue
		}
		if path != "" {
			nm = path + "." + nm
		}
		b.WriteString("-" + strcase.ToKebab(nm) + "\n")
		desc, ok := f.Tag.Lookup("desc")
		if ok && desc != "" {
			b.WriteString("\t")
			b.WriteString(desc)
			def, ok := f.Tag.Lookup("def")
			if ok && def != "" {
				b.WriteString(fmt.Sprintf(" (default %s)", def))
			}
		}
		b.WriteString("\n")
	}
}
