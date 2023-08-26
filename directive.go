// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package directive implements simple, standardized, and scalable parsing of Go comment directives.
package directive

import (
	"sort"
	"strings"
)

// Directive represents a comment directive
// that has been parsed or created in code.
type Directive struct {

	// Source is the source string of the
	// comment directive.
	Source string

	// Tool is the name of the tool that
	// the directive is for.
	Tool string

	// Directive is the actual directive
	// string that is placed after the
	// name of the tool and a colon.
	Directive string

	// Args are the positional arguments
	// passed to the directive
	Args []string

	// NameValue are a map of name-value
	// arguments given in the
	// form `name=value`.
	NameValue map[string]string
}

// Parse parses the given comment string and returns
// any [Directive] inside it. If no such directive is
// found, it returns nil. Directives are of the form:
// `//tool:directive arg0 key0=value0 arg1 key1=value1`
// (the two slashes are optional, and the positional
// and key-value arguments can be in any order).
func Parse(comment string) *Directive {
	dir := &Directive{}
	before, after, found := strings.Cut(comment, ":")
	if !found {
		return nil
	}
	dir.Source = comment
	before = strings.TrimPrefix(before, "//")
	dir.Tool = before
	dir.Args = []string{}
	dir.NameValue = map[string]string{}
	fields := strings.Fields(after)
	for i, field := range fields {
		if i == 0 {
			dir.Directive = field
			continue
		}
		before, after, found := strings.Cut(field, "=")
		if found {
			dir.NameValue[before] = after
		} else {
			dir.Args = append(dir.Args, before)
		}
	}
	return dir
}

// String returns the directive as a
// formatted string suitable for use in
// code. It puts the positional arguments
// before the name-value arguments, and it
// includes two slashes (`//`) at the start.
// The output of String is deterministic
// because it sorts the name-value map.
func (d *Directive) String() string {
	if d == nil {
		return "<nil>"
	}
	res := "//" + d.Tool + ":" + d.Directive
	for _, arg := range d.Args {
		res += " " + arg
	}
	keys := make([]string, 0, len(d.NameValue))
	for key := range d.NameValue {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		res += " " + key + "=" + d.NameValue[key]
	}
	return res
}
