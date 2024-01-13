// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import (
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

func (d Directive) GoString() string { return StructGoString(d) }
