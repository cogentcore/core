// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

// Func represents a global function.
type Func struct {
	// Name is the fully qualified name of the function
	// (eg: cogentcore.org/core/core.NewButton)
	Name string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives are the parsed comment directives
	Directives []Directive

	// Args are the names of the arguments to the function
	Args []string

	// Returns are the names of the return values of the function
	Returns []string

	// ID is the unique function ID number
	ID uint64
}

func (f Func) GoString() string { return StructGoString(f) }

// Method represents a method.
type Method struct {
	// Name is the name of the method (eg: NewChild)
	Name string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives are the parsed comment directives
	Directives []Directive

	// Args are the names of the arguments to the function
	Args []string

	// Returns are the names of the return values of the function
	Returns []string
}

func (m Method) GoString() string { return StructGoString(m) }
