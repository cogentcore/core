// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"github.com/goki/gi/mat32"
	"github.com/goki/ki/kit"
)

// Input represents an input to a program, i.e., a Vertex Buffer Object
// it is created by Program.AddInput
type Input interface {
	// Name returns the name of the input (i.e., as it is referred to in the shader program)
	Name() string

	// Type returns the input data type
	Type() InputType

	// Role returns the functional role of this input
	Role() InputRoles

	// ByteOffset returns the starting offset of this item relative to start of InputBuffer
	ByteOffset() int32

	// Stride returns the stride of this item within InputBuffer
	Stride() uint32

	// Len returns the number of elements of this input
	Len() int32

	// SetLen sets the number of elements of this input
	SetLen(ln int32)

	// Handle returns the unique handle for this input within the program where it is used
	Handle() uint32
}

// InputBuffer represents a buffer (i.e., VBO) with multiple Input elements
type InputBuffer interface {
	// Name returns the name of this buffer
	Name() string

	// Usage returns whether this is dynamic or static etc
	Usage() uint32

	// SetUsage sets the usage of the buffer
	SetUsage(usg uint32)

	// AddInput adds an Input that reads from this buffer.
	// Inputs are created in a Program, and bound to this buffer here.
	AddInput(in Input)

	// Handle returns the unique handle for this buffer
	Handle() uint32

	// SetData sets the data in the buffer
	SetData(data mat32.ArrayF32)
}

// InputRoles are the functional roles of an input
type InputRoles int32

const (
	Undefined InputRoles = iota
	VertexPosition
	VertexNormal
	VertexTangent
	VertexColor
	VertexTexcoord
	VertexTexcoord2
	SkinWeight
	SkinIndex
	InputRolesN
)

//go:generate stringer -type=InputRoles

var KiT_InputRoles = kit.Enums.AddEnum(InputRolesN, false, nil)
