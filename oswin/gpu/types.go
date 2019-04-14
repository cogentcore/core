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
	UndefType Types = iota
	Bool
	Int
	UInt
	Float32
	Float64
	TypesN
)

//go:generate stringer -type=Types

var KiT_Types = kit.Enums.AddEnum(TypesN, false, nil)

// GLSL type names
var TypeNames = map[Types]string{
	UndefType: "none",
	Bool:      "bool",
	Int:       "int",
	UInt:      "uint",
	Float32:   "float",
	Float64:   "double",
}

// TypeBytes returns number of bytes for given type -- 4 except Float64 = 8
func TypeBytes(tp Types) int {
	if tp == Float64 {
		return 8
	}
	return 4
}

// UniType represents a fully-specified GPU uniform type, including vectors and matricies
type UniType struct {
	Type Types `desc:"data type"`
	Vec  int   `desc:"if a vector, this is the length of the vector, 0 for scalar (valid values are 2,3,4)"`
	Mat  int   `desc:"square matrix dimensions, if a matrix (valid values are 3,4)"`
}

// Name returns the full GLSL type name for the type
func (ty *UniType) Name() string {
	if ty.Vec == 0 && ty.Mat == 0 {
		return TypeNames[ty.Type]
	}
	pfx := TypeNames[ty.Type][0:1]
	if ty.Type == Float32 {
		pfx = ""
	}
	if ty.Vec > 0 {
		return fmt.Sprintf("%svec%d", pfx, ty.Vec)
	} else {
		return fmt.Sprintf("%smat%d", pfx, ty.Mat)
	}
}

// Bytes returns number of bytes taken up by this element, in std140 format (including padding)
// https://learnopengl.com/Advanced-OpenGL/Advanced-GLSL
func (ty *UniType) Bytes() int {
	n := TypeBytes(ty.Type)
	if ty.Vec == 0 && ty.Mat == 0 {
		return n
	}
	if ty.Vec > 0 {
		if ty.Vec <= 2 {
			return 2 * n
		}
		return 4 * n
	}
	return ty.Mat * 4 * n
}

// VectorType represents a fully-specified GPU vector type, e.g., for inputs / outputs
// to shader programs
type VectorType struct {
	Type Types `desc:"data type"`
	Vec  int   `desc:"length of vector (valid values are 2,3,4)"`
}

// Bytes returns number of bytes per Vector element (len * 4 basically)
func (ty *VectorType) Bytes() int {
	n := TypeBytes(ty.Type)
	return n * ty.Vec
}
