// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"

	"github.com/goki/ki/kit"
)

// See: https://www.khronos.org/opengl/wiki/Data_Type_(GLSL)

// Types is a list of GPU data types
type Types int32

const (
	NoType Types = iota
	Bool
	Int
	UInt
	Float32
	Float64
	TypesN
)

//go:generate stringer -type=Types

var KiT_Types = kit.Enums.AddEnum(TypesN, false, nil)

var TypeNames = map[Type]string{
	NoType:  "none",
	Bool:    "bool",
	Int:     "int",
	UInt:    "uint",
	Float32: "float",
	Float64: "double",
}

// UniType represents a fully-specified GPU uniform type, including vectors and matricies
type UniType struct {
	Type           Types `desc:"data type"`
	Vec            int   `desc:"if a vector, this is the length of the vector, 0 for scalar (valid values are 2,3,4)"`
	MatCol, MatRow int   `desc:"matrix dimensions, if a matrix (valid values are 2,3,4)"`
}

// Name returns the full GLSL type name for the type
func (ty *UniType) Name() {
	if ty.Vec == 0 && ty.MatCol == 0 {
		return TypeNames[ty.Type]
	}
	pfx := TypeNames[ty.Type][0]
	if ty.Type == Float32 {
		pfx = ""
	}
	if ty.Vec > 0 {
		return fmt.Sprintf("%svec%d", pfx, ty.Vec)
	} else if ty.MatCol == ty.MatRow {
		return fmt.Sprintf("%smat%d", pfx, ty.MatCol)
	} else {
		return fmt.Sprintf("%smat%dx%d", pfx, ty.MatCol, ty.MatRow)
	}
}

// InputType represents a fully-specified GPU vertex input type, including vectors and matricies
type InputType struct {
	Type Types `desc:"data type"`
	Len  int   `desc:"length of vector (valid values are 2,3,4)"`
}
