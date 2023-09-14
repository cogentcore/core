// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import (
	"reflect"
)

// Type represents a type
type Type struct {
	// Name is the fully-package-path-qualified name of the type (eg: goki.dev/gi/v2/gi.Button)
	Name string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives has the parsed comment directives
	Directives Directives

	// unique type ID number
	ID uint64 `desc:"unique type ID number"`

	// Methods are available for all types
	Methods *Methods

	// Embeded fields for struct types
	Embeds *Fields

	// All embedded fields (including nested ones) for struct types;
	// not set by gtigen but can be set manually by outside packages
	AllEmbeds *Field

	// Fields for struct types
	Fields *Fields

	// instance of the type
	Instance any `desc:"instance of the type"`
}

// ReflectType returns the [reflect.Type] of a given Ki Type
func (tp *Type) ReflectType() reflect.Type {
	return reflect.TypeOf(tp.Instance).Elem()
}
