// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

// Directive represents a comment directive in the format:
//
//	//tool:directive args...
type Directive struct {
	Tool      string
	Directive string
	Args      []string
}

// this is helpful for literals

type Directives []*Directive

// todo: methods for returning all directives for given tool name
