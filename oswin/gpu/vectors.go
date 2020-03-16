// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Vectors manages arrays of vectors that are processed as inputs to a shader program
// and received as outputs from compute shaders.  i.e., a Vertex Buffer Object in OpenGL.
// It is created by Program.AddInputs, .AddOutputs, and stores the Handle into that program's
// variable.  This handle is then bound to a buffer in VectorsBuffer.
type Vectors interface {
	// Name returns the name of the vectors (i.e., as it is referred to in the shader program)
	Name() string

	// Type returns the vector data type
	Type() VectorType

	// Role returns the functional role of these vectors
	Role() VectorRoles

	// Handle returns the unique handle for these vectors within the program where it is used
	// Can also be specified using layout(location = X) specifier.
	Handle() uint32

	// Set sets all the parameters of the Vectors, and flags it as init -- when
	// created for predefined locations.
	Set(name string, handle uint32, typ VectorType, role VectorRoles)
}

// https://www.khronos.org/opengl/wiki/Vertex_Specification_Best_Practices
// https://www.khronos.org/opengl/wiki/Vertex_Specification#Vertex_Buffer_Object
// critical points:
// 1. need a VAO at start to hold active buffers
// 2. only one buffer of each type can be active at a time (ARRAY, ELEMENT = index)
// 3. buffer attributes must be configured in context of actual buffer being active
//    using glVertexAttribPtr
// 4. thus all these steps are done at same point at each render, just prior to draw
//    this is the render step.

// VectorsBuffer represents a buffer with multiple Vectors elements, which
// can be either interleaved (contiguous from the start only) or appended seqeuentially.
// All elements must be Float32, not Float64!  Need a different buffer type that handles 64bit.
// It is created in BufferMgr.AddVectorsBuffer -- the Mgr is essential for managing buffers.
// The buffer maintains its own internal memory storage (mat32.ArrayF32)
// which can be operated upon or set from external sources.
type VectorsBuffer interface {
	// Usage returns whether this is dynamic or static etc
	Usage() VectorUsages

	// SetUsage sets the usage of the buffer
	SetUsage(usg VectorUsages)

	// AddVectors adds a Vectors to this buffer, all interleaved vectors
	// must be added first, before any non-interleaved (error will be logged if not).
	// Vectors are created in a Program, and connected to this buffer here.
	// All Vectors in a given Program must be stored in a SINGLE buffer.
	// Add all Vectors before setting the length, which then computes offset and strides
	// for each vector.
	AddVectors(vec Vectors, interleave bool)

	// NumVectors returns number of vectors in the buffer
	NumVectors() int

	// Vectors returns a list (slice) of all the vectors in the buffer, in order.
	Vectors() []Vectors

	// VectorsByName returns given vectors by name.
	// Returns nil if not found (error auto logged)
	VectorsByName(name string) Vectors

	// VectorsByRole returns given vectors by role.
	// Returns nil if not found (error auto logged)
	VectorsByRole(role VectorRoles) Vectors

	// SetLen sets the number of elements in the buffer -- must be same number for each
	// Vectors type in buffer.  Also triggers computation of offsets and strides for each
	// vector -- call after having added all vectors.
	SetLen(ln int)

	// Len returns the number of elements in the buffer.
	Len() int

	// ByteOffset returns the starting offset of given Vectors in buffer.
	// Only valid after SetLen has been called.
	ByteOffset(vec Vectors) int

	// Offset returns the float element wise starting offset of given Vectors (ByteOffset / 4).
	// Only valid after SetLen has been called.
	Offset(vec Vectors) int

	// Stride returns the float element stride of given Vectors in buffer
	Stride(vec Vectors) int

	// ByteStride returns the byte stride of given Vectors in buffer (Stride * 4)
	ByteStride(vec Vectors) int

	// SetAllData sets all of the data in the buffer copying from given source
	SetAllData(data mat32.ArrayF32)

	// AllData returns the raw buffer data. This is the pointer to the internal
	// data -- if you modify it, you modify the internal data!  copy first if needed.
	AllData() mat32.ArrayF32

	// SetVecData sets data for given Vectors -- handles interleaving etc
	SetVecData(vec Vectors, data mat32.ArrayF32)

	// VecData returns data for given Vectors -- this is a copy for interleaved data
	// and a direct sub-slice for non-interleaved.
	VecData(vec Vectors) mat32.ArrayF32

	// Vec3Func iterates over all values of given vec3 Vectors
	// and calls the specified callback function with a pointer to each item as a Vec3.
	// Modifications to vec will be applied to the buffer at each iteration.
	// The callback function returns false to break or true to continue.
	Vec3Func(vec Vectors, fun func(vec *mat32.Vec3) bool)

	// Activate binds buffer as active one, and configures it per all existing settings
	Activate()

	// IsActive returns true if buffer has already been Activate'd
	// and thus exists on the GPU
	IsActive() bool

	// Handle returns the unique handle for this buffer -- only valid after Activate()
	Handle() uint32

	// Transfer transfers data to GPU -- Activate must have been called with no other
	// such buffers activated in between.  Automatically uses re-specification
	// strategy per: https://www.khronos.org/opengl/wiki/Buffer_Object_Streaming
	// so it is safe if buffer was still being used from prior GL rendering call.
	Transfer()

	// TransferVec transfers only data for given vector to GPU -- only valid
	// if Activate() and Transfer() have been called already, and only for
	// non-interleaved vectors.
	TransferVec(vec Vectors)

	// Delete deletes the GPU resources associated with this buffer
	// (requires Activate to re-establish a new one).
	// Should be called prior to Go object being deleted
	// (ref counting can be done externally).
	Delete()

	// DeleteAllVectors deletes all Vectors defined for this buffer (calls Delete first)
	DeleteAllVectors()

	// DeleteVectorsByName deletes Vectors of given name (calls Delete first)
	DeleteVectorsByName(name string)

	// DeleteVectorsByRole deletes Vectors of given role (calls Delete first)
	DeleteVectorsByRole(role VectorRoles)

	// GPUUsage returns the GPU vector usage id for given usage
	GPUUsage(usg VectorUsages) uint32
}

// VectorRoles are the functional roles of vectors
type VectorRoles int32

const (
	UndefRole VectorRoles = iota
	VertexPosition
	VertexNormal
	VertexTangent
	VertexColor
	VertexTexcoord
	VertexTexcoord2
	SkinWeight
	SkinIndex
	VectorRolesN
)

//go:generate stringer -type=VectorRoles

var KiT_VectorRoles = kit.Enums.AddEnum(VectorRolesN, kit.NotBitFlag, nil)

// see: https://computergraphics.stackexchange.com/questions/5712/gl-static-draw-vs-gl-dynamic-draw-vs-gl-stream-draw-does-it-matter
// https://www.khronos.org/registry/OpenGL-Refpages/gl4/html/glBufferData.xhtml
// http://docs.gl/gl4/glBufferStorage (new more precise version that is only avail in 4.4)

// VectorUsages are the usage hints for vector buffers
type VectorUsages int32

const (
	// The data store contents will be modified once and used at most a few times.
	// The data store contents are modified by the application, and used as the source for GL drawing and image specification commands.
	StreamDraw VectorUsages = iota

	// The data store contents will be modified once and used at most a few times.
	// The data store contents are modified by reading data from the GL, and used to return that data when queried by the application.
	StreamRead

	// The data store contents will be modified once and used at most a few times.
	// The data store contents are modified by reading data from the GL, and used as the source for GL drawing and image specification commands.
	StreamCopy

	// The data store contents will be modified once and used many times.
	// The data store contents are modified by the application, and used as the source for GL drawing and image specification commands.
	StaticDraw

	// The data store contents will be modified once and used many times.
	// The data store contents are modified by reading data from the GL, and used to return that data when queried by the application.
	StaticRead

	// The data store contents will be modified once and used many times.
	// The data store contents are modified by reading data from the GL, and used as the source for GL drawing and image specification commands.
	StaticCopy

	// The data store contents will be modified repeatedly and used many times.
	// The data store contents are modified by the application, and used as the source for GL drawing and image specification commands.
	DynamicDraw

	// The data store contents will be modified repeatedly and used many times.
	// The data store contents are modified by reading data from the GL, and used to return that data when queried by the application.
	DynamicRead

	// The data store contents will be modified repeatedly and used many times.
	// The data store contents are modified by reading data from the GL, and used as the source for GL drawing and image specification commands.
	DynamicCopy

	VectorUsagesN
)

//go:generate stringer -type=VectorUsages

var KiT_VectorUsages = kit.Enums.AddEnum(VectorUsagesN, kit.NotBitFlag, nil)
