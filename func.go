// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import "goki.dev/ordmap"

// Func represents a global function
type Func struct {
	Name string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives has the parsed comment directives
	Directives Directives

	// Args are arguments to the method
	Args *Fields

	// Returns are return values of the method
	Returns *Fields

	// unique type ID number
	ID uint64 `desc:"unique type ID number"`
}

// Method represents a method
type Method struct {
	Name string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives has the parsed comment directives
	Directives Directives

	// Args are arguments to the function
	Args *Fields

	// Returns are return values of the function
	Returns *Fields
}

// Methods represents a set of multiple [Method] objects
type Methods = ordmap.Map[string, *Method]
