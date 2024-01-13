// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import (
	"fmt"
	"reflect"
	"strings"
)

// Directive represents a comment directive in the format:
//
//	//tool:directive args...
type Directive struct {
	Tool      string
	Directive string
	Args      []string
}

// String returns a string representation of the directive
// in the format:
//
//	//tool:directive args...
func (d Directive) String() string {
	return "//" + d.Tool + ":" + d.Directive + " " + strings.Join(d.Args, " ")
}

// GoString returns the directive as Go code.
func (d Directive) GoString() string {
	return StructGoString(d)
}

// StructGoString creates a GoString for the given struct while omitting
// any zero values.
func StructGoString(str any) string {
	s := reflect.ValueOf(str)
	typ := s.Type()
	strs := []string{}
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if f.IsZero() {
			continue
		}
		nm := typ.Field(i).Name
		strs = append(strs, fmt.Sprintf("%s: %#v", nm, f))

	}
	return "{" + strings.Join(strs, ", ") + "}"
}

// this is helpful for literals

type Directives []Directive

// GoString returns the directives as Go code.
func (d Directives) GoString() string {
	res := "gti.Directives{\n"
	for _, dir := range d {
		res += dir.GoString()
		res += ",\n"
	}
	res += "}"
	return res
}

// ForTool returns all of the directives in these
// directives that have the given tool name.
func (d Directives) ForTool(tool string) Directives {
	res := Directives{}
	for _, dir := range d {
		if dir.Tool == tool {
			res = append(res, dir)
		}
	}
	return res
}
