// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"unsafe"

	"github.com/goki/pi/syms"
)

var BuiltinTypes syms.TypeMap

// InstallBuiltinTypes initializes the BuiltinTypes map
func InstallBuiltinTypes() {
	if len(BuiltinTypes) != 0 {
		return
	}
	for _, tk := range BuiltinTypeKind {
		ty := syms.NewType(tk.Name, tk.Kind)
		ty.Size = []int{tk.Size}
		BuiltinTypes.Add(ty)
	}
}

// BuiltinTypeKind are the type names and kinds for builtin Go primitive types
// (i.e., those with names)
var BuiltinTypeKind = []syms.TypeKindSize{
	{"int", syms.Int, int(unsafe.Sizeof(int(0)))},
	{"int8", syms.Int8, 1},
	{"int16", syms.Int16, 2},
	{"int32", syms.Int32, 4},
	{"int64", syms.Int64, 8},

	{"uint", syms.Uint, int(unsafe.Sizeof(uint(0)))},
	{"uint8", syms.Uint8, 1},
	{"uint16", syms.Uint16, 2},
	{"uint32", syms.Uint32, 4},
	{"uint64", syms.Uint64, 8},
	{"uintptr", syms.Uintptr, 8},

	{"byte", syms.Uint8, 1},
	{"rune", syms.Int32, 4},

	{"float32", syms.Float32, 4},
	{"float64", syms.Float64, 8},

	{"complex64", syms.Complex64, 8},
	{"complex128", syms.Complex128, 16},

	{"bool", syms.Bool, 1},

	{"string", syms.String, 0},

	{"error", syms.Interface, 0},

	{"struct{}", syms.Struct, 0},
	{"interface{}", syms.Interface, 0},
}
