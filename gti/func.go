// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

// Func represents a global function
type Func struct {
	// Name is the fully-qualified name of the function
	// (eg: goki.dev/gi.NewButton)
	Name string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives has the parsed comment directives
	Directives Directives

	// Args are arguments to the function
	Args Fields

	// Returns are return values of the function
	Returns Fields

	// ID is the unique type ID number
	ID uint64
}

func (f Func) GoString() string { return StructGoString(f) }

// Method represents a method
type Method struct {
	// Name is the name of the method (eg: NewChild)
	Name string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives has the parsed comment directives
	Directives Directives

	// Args are arguments to the method
	Args Fields

	// Returns are return values of the method
	Returns Fields
}

func (m Method) GoString() string { return StructGoString(m) }

// Methods represents a set of multiple [Method] objects
type Methods = []Method
