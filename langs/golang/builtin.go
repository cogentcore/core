// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import "github.com/goki/pi/syms"

var BuiltinTypes syms.TypeMap

// InstallBuiltinTypes initializes the BuiltinTypes map
func InstallBuiltinTypes() {
	if len(BuiltinTypes) != 0 {
		return
	}
	for _, tk := range BuiltinTypeKind {
		BuiltinTypes.Add(syms.NewType(tk.Name, tk.Kind))
	}
}

// BuiltinTypeKind are the type names and kinds for builtin Go primitive types
// (i.e., those with names)
var BuiltinTypeKind = []syms.TypeKind{
	{"int", syms.Int},
	{"int8", syms.Int8},
	{"int16", syms.Int16},
	{"int32", syms.Int32},
	{"int64", syms.Int64},

	{"uint", syms.Uint},
	{"uint8", syms.Uint8},
	{"uint16", syms.Uint16},
	{"uint32", syms.Uint32},
	{"uint64", syms.Uint64},
	{"uintptr", syms.Uintptr},

	{"byte", syms.Uint8},
	{"rune", syms.Int32},

	{"float32", syms.Float32},
	{"float64", syms.Float64},

	{"complex64", syms.Complex64},
	{"complex128", syms.Complex128},

	{"bool", syms.Bool},

	{"string", syms.String},

	{"error", syms.Interface},
}
