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
	return fmt.Sprintf("\n&%#v", *d)
}

// this is helpful for literals

type Directives []*Directive

// todo: methods for returning all directives for given tool name
