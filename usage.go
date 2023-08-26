// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package econfig

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/goki/ki/kit"
)

// Usage returns the usage string for args based on given Config object
func Usage(cfg any) string {
	var b strings.Builder
	b.WriteString("The following command-line arguments set fields on the Config struct.\n")
	b.WriteString("args are case insensitive and kebab-case or snake_case also works\n")
	b.WriteString("most can be used without nesting path (e.g. -nepochs instead of -run.nepochs)\n")
	b.WriteString("\n")
	b.WriteString("-help or -h\tshow available command-line arguments and exit\n")
	b.WriteString("-config or -cfg\tspecify filename for loading Config settings\n")
	b.WriteString("\n")
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
