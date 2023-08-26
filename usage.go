// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/goki/ki/kit"
)

// Usage returns the usage string for args based on given Config object
func Usage(cfg any) string {
	var b strings.Builder
	b.WriteString(AppAbout)
	b.WriteString("\n\n")
	b.WriteString("The following flags are supported. Flags are case-insensitive and\n")
	b.WriteString("can be in CamelCase, snake_case, or kebab-case. Also, there can be\n")
	b.WriteString("one or two leading dashes. Most flags can be used without nesting\n")
	b.WriteString("paths (e.g. -target instead of -build.target)\n\n")
	b.WriteString("-help or -h\n\tshow this usage message and exit\n")
	b.WriteString("-config or -cfg\n\tthe filename to load configuration options from\n")
	usageStruct(cfg, "", &b)
	return b.String()
}

// usageStruct adds usage info to given strings.Builder
func usageStruct(obj any, path string, b *strings.Builder) {
	typ := kit.NonPtrType(reflect.TypeOf(obj))
	val := kit.NonPtrValue(reflect.ValueOf(obj))
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fv := val.Field(i)
		if kit.NonPtrType(f.Type).Kind() == reflect.Struct {
			nwPath := f.Name
			if path != "" {
				nwPath = path + "." + nwPath
			}
			usageStruct(kit.PtrValue(fv).Interface(), nwPath, b)
			continue
		}
		nm := f.Name
		if nm == "Includes" {
			continue
		}
		if path != "" {
			nm = path + "." + nm
		}
		b.WriteString(fmt.Sprintf("-%s\n", nm))
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
