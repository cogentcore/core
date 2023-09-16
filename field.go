// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"reflect"

	"goki.dev/ordmap"
)

// Field represents a struct field in a configuration object.
// It is passed around in flag parsing functions, but it should
// not typically be used by end-user code going through the
// standard Run/Config/SetFromArgs API.
type Field struct {
	// Field is the reflect struct field object for this field
	Field reflect.StructField
	// Value is the reflect value of the settable pointer to this field
	Value reflect.Value
	// Name is the fully qualified, nested name of this field (eg: A.B.C).
	// It is as it appears in code, and is NOT transformed something like kebab-case.
	Name string
}

// Fields is a simple type alias for an ordered map of [Field] objects.
type Fields = *ordmap.Map[string, *Field]
