// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import (
	"fmt"
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
func (d *Directive) String() string {
	return "//" + d.Tool + ":" + d.Directive + " " + strings.Join(d.Args, " ")
}

// GoString returns the directive as Go code.
func (d *Directive) GoString() string {
	return fmt.Sprintf("&%#v", *d)
}

// this is helpful for literals

type Directives []*Directive

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

// TODO: do we need this?

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
