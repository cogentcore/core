// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import (
	"reflect"

	"goki.dev/ordmap"
)

// Field represents a field or embed in a struct,
// or an argument or return value of a function.
type Field struct {
	Name string

	// Type has the fully-package-path-qualified name of the type,
	// which can be used to look up the type in the Types registry
	Type string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives has the parsed comment directives
	Directives Directives

	// Tag, if this field is part of a struct, contains the struct
	// tag for it.
	Tag reflect.StructTag
}

// Fields represents a set of multiple [Field] objects
type Fields = ordmap.Map[string, *Field]
