// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import "goki.dev/ordmap"

// Func represents a global function
type Func struct {
	Name string

	// Comment has all of the comment info as one string
	// with directives removed.
	Comment string

	// Directives has the parsed comment directives
	Directives Directives

	// Args are arguments to the method
	Args *ordmap.Map[string, *Arg]

	// Returns are return values of the function
	Returns *ordmap.Map[string, *Arg]

	// unique type ID number
	ID uint64 `desc:"unique type ID number"`
}

// Method represents a method
type Method struct {
	Name string

	// Comment has all of the comment info as one string
	// with directives removed.
	Comment string

	// Directives has the parsed comment directives
	Directives Directives

	// Args are arguments to the method
	Args *ordmap.Map[string, *Arg]

	// Returns are return values of the function
	Returns *ordmap.Map[string, *Arg]
}
