// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"
	"unsafe"

	"github.com/goki/mat32"
)

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
// It is created in VecIdxs.AddVectorsBuffer -- the Mgr is essential for managing buffers.
// The buffer maintains its own internal memory storage (mat32.ArrayF32)
// which can be operated upon or set from external sources.
// Note: all arrangement data is in *float* units, not *byte* units -- multiply * 4 to get bytes.
// All transfer to GPU and memory management is handled at the Memory level!
type VectorsBuffer struct {
	init      bool
	trans     bool // was buffer already transferred up to device yet?
	mod       bool // were vector params modified at all?
	handle    uint32
	usage     VectorUsages
	vecs      []*Var
	stride    int            // number of float elements stride for interleaved (*not in bytes*)
	offs      []int          // float offsets per vector in floats  (*not in bytes*)
	nInter    int            // number of interleaved
	ln        int            // number of elements per vector
	totLn     int            // total length of buffer in floats (*not in bytes*)
	buff      mat32.ArrayF32 `desc:"local buffer that has full data as written by user code"`
	BuffAlloc BuffAlloc      `desc:"buffer allocation in terms of vulkan device-side GPU buffer, for vk.CmdBindVertexBuffers"`
}

// AddVectors adds a Vectors to this buffer, all interleaved Vectors
// must be added first, before any non-interleaved (error will be logged if not).
// Vectors are created in a Program, and connected to this buffer here.
// All Vectors in a given Program must be stored in a SINGLE buffer.
// Add all Vectors before setting the length, which then computes offset and strides
// for each vector.
func (vb *VectorsBuffer) AddVectors(vec *Var, interleave bool) {
	vb.trans = false
	vb.mod = true
	v := vec.(*Var)
	if v.typ.Type == Float64 {
		log.Printf("glos.VectorsBuffer AddVectors: Float64 not supported for this buffer type\n")
		return
	}

	ncur := len(vb.vecs)
	vb.vecs = append(vb.vecs, v)
	if interleave {
		if ncur != vb.nInter {
			log.Printf("glos.VectorsBuffer AddVectors: all interleaved must be added together at start\n")
		} else {
			vb.nInter++
		}
	}
}

// NumVectors returns number of vectors in the buffer
func (vb *VectorsBuffer) NumVectors() int {
	return len(vb.vecs)
}

// Vectors returns a list (slice) of all the Vectors in the buffer, in order.
func (vb *VectorsBuffer) Var() []*Var {
	vecs := make([]*Var, len(vb.vecs))
	for i := range vb.vecs {
		vecs[i] = vb.vecs[i]
	}
	return vecs
}

// VectorsByName returns given Vectors by name.
// Returns nil if not found (error auto logged)
func (vb *VectorsBuffer) VectorsByName(name string) *Var {
	for _, v := range vb.vecs {
		if v.name == name {
			return v
		}
	}
	log.Printf("glos.VectorsBuffer VectorsByName: name %v not found\n", name)
	return nil
}

// VectorsByRole returns given Vectors by role.
// Returns nil if not found (error auto logged)
func (vb *VectorsBuffer) VectorsByRole(role VectorRoles) *Var {
	for _, v := range vb.vecs {
		if v.role == role {
			return v
		}
	}
	log.Printf("glos.VectorsBuffer VectorsByRole: role %s not found\n", role)
	return nil
}

// updates all vec info
func (vb *VectorsBuffer) updtVecs() {
	vb.stride = 0
	vb.totLn = 0
	sz := len(vb.vecs)
	if sz == 0 {
		return
	}
	if len(vb.offs) != sz {
		vb.offs = make([]int, sz)
	}
	if vb.nInter > 0 {
		str := 0
		for i := 0; i < vb.nInter; i++ {
			v := vb.vecs[i]
			vb.offs[i] = str
			str += v.typ.Vec
		}
		vb.stride = str
		vb.totLn = vb.stride * vb.ln
	}
	off := vb.totLn
	for i := vb.nInter; i < sz; i++ {
		v := vb.vecs[i]
		vb.offs[i] = off
		off += vb.ln * v.typ.Vec
	}
	vb.totLn = off
	if len(vb.buff) != vb.totLn {
		vb.buff = make(mat32.ArrayF32, vb.totLn)
	}
}

// SetLen sets the number of elements in the buffer -- must be same number for each
// Vectors type in buffer.  Also triggers computation of offsets and strides for each
// vector -- call after having added all Vectors.
func (vb *VectorsBuffer) SetLen(ln int) {
	if vb.ln == ln {
		return
	}
	vb.trans = false
	vb.mod = true
	vb.ln = ln
	vb.updtVecs()
}

// Len returns the number of elements in the buffer.
func (vb *VectorsBuffer) Len() int {
	return vb.ln
}

// MemSize returns total number of bytes of memory needed
func (vb *VectorsBuffer) MemSize() int {
	return vb.totLn * 4
}

func (vb *VectorsBuffer) vec(vec *Var) (int, *Var) {
	for i, v := range vb.vecs {
		if v == vec {
			return i, v
		}
	}
	log.Printf("glos.VectorsBuffer: vector named: %s not found\n", vec.Name())
	return -1, nil
}

// ByteOffset returns the starting offset of given Vectors in buffer
func (vb *VectorsBuffer) ByteOffset(vec *Var) int {
	i, _ := vb.vec(vec)
	if i >= 0 {
		return vb.offs[i] * 4
	}
	return 0
}

// Offset returns the float element wise starting offset of given Vectors in buffer
func (vb *VectorsBuffer) Offset(vec *Var) int {
	return vb.Offset(vec)
}

// Stride returns the float-element-wise stride of given Vectors
func (vb *VectorsBuffer) Stride(vec *Var) int {
	i, _ := vb.vec(vec)
	if i >= 0 {
		if i < vb.nInter {
			return vb.stride
		}
		return 0
	}
	return 0
}

// ByteStride returns the byte-wise stride of given Vectors
func (vb *VectorsBuffer) ByteStride(vec *Var) int {
	return vb.Stride(vec) * 4
}

// SetAllData sets all of the data in the buffer copying from given source
func (vb *VectorsBuffer) SetAllData(data mat32.ArrayF32) {
	copy(vb.buff, data)
}

// AllData returns the raw buffer data. This is the pointer to the internal
// data -- if you modify it, you modify the internal data!  copy first if needed.
func (vb *VectorsBuffer) AllData() mat32.ArrayF32 {
	return vb.buff
}

// SetVecData sets data for given Vectors -- handles interleaving etc
func (vb *VectorsBuffer) SetVecData(vec *Var, data mat32.ArrayF32) {
	i, v := vb.vec(vec)
	if i < 0 {
		return
	}
	off := vb.offs[i]
	els := v.typ.Vec
	str := els
	if i < vb.nInter {
		str = vb.stride
	}
	for i := 0; i < vb.ln; i++ {
		bidx := off + i*str
		sidx := i * els
		if sidx >= len(data) {
			break
		}
		for j := 0; j < els; j++ {
			vb.buff[bidx+j] = data[sidx+j]
		}
	}
}

// VecData returns data for given Vectors -- this is a copy for interleaved data
// and a direct sub-slice for non-interleaved.
func (vb *VectorsBuffer) VecData(vec *Var) mat32.ArrayF32 {
	i, v := vb.vec(vec)
	if i < 0 {
		return nil
	}
	off := vb.offs[i]
	els := v.typ.Vec
	sz := els * vb.ln
	if i >= vb.nInter {
		return vb.buff[off : off+sz]
	}
	str := vb.stride
	rv := make(mat32.ArrayF32, sz)
	for i := 0; i < vb.ln; i++ {
		bidx := off + i*str
		sidx := i * els
		for j := 0; j < els; j++ {
			rv[sidx+j] = vb.buff[bidx+j]
		}
	}
	return rv
}

// Vec3Func iterates over all values of given vec3 vector
// and calls the specified callback function with a pointer to each item as a Vec3.
// Modifications to vec will be applied to the buffer at each iteration.
// The callback function returns false to break or true to continue.
func (vb *VectorsBuffer) Vec3Func(vec *Var, fun func(vec *mat32.Vec3) bool) {
	i, v := vb.vec(vec)
	if i < 0 {
		return
	}
	off := vb.offs[i]
	els := v.typ.Vec
	str := els
	if i < vb.nInter {
		str = vb.stride
	}
	var v3 mat32.Vec3
	for i := 0; i < vb.ln; i++ {
		bidx := off + i*str
		vb.buff.GetVec3(bidx, &v3)
		cont := fun(&v3)
		vb.buff.SetVec3(bidx, v3)
		if !cont {
			break
		}
	}
}

// float: VK_FORMAT_R32_SFLOAT = FormatR32Sfloat
// vec2: VK_FORMAT_R32G32_SFLOAT
// vec3: VK_FORMAT_R32G32B32_SFLOAT
// vec4: VK_FORMAT_R32G32B32A32_SFLOAT

// Alloc allocates this buffer from overall buffer memory at given offset
func (vb *VectorsBuffer) Alloc(mm *Memory, offset int) int {
	if vb.init {
		fmt.Printf("attempting to allocate already-initialized!\n") // todo: shouldn't happen..
	}
	vb.updtVecs() // make sure

	sz := vb.MemSize()
	vb.BuffAlloc.Set(mm, offset, sz)
	vb.init = true
	return sz
}

// Free nulls the Allocation
func (vb *VectorsBuffer) Free() {
	vb.init = false
	vb.BuffAlloc.Free()
}

// CopyBuffToStaging copies all of the buffer source data into the CPU side staging buffer.
// this does not check for changes -- use for initial configuration.
func (vb *VectorsBuffer) CopyBuffToStaging(bufPtr unsafe.Pointer) {
	BuffMemCopy(&vb.BuffAlloc, bufPtr, unsafe.Pointer(&(vb.buff[0])))
	vb.mod = false
}

// SyncBuffToStaging copies all of the buffer source data into the CPU side staging buffer.
// only if data has changed as indicated by the mod flag.  resets flag.
func (vb *VectorsBuffer) SyncBuffToStaging(bufPtr unsafe.Pointer) bool {
	if !vb.mod {
		return false
	}
	BuffMemCopy(&vb.BuffAlloc, bufPtr, unsafe.Pointer(&(vb.buff[0])))
	vb.mod = false
	return true
}

// Activate: remove
func (vb *VectorsBuffer) Activate() {
}

// IsActive returns true if buffer has already been Activate'd
// and thus exists on the GPU
func (vb *VectorsBuffer) IsActive() bool {
	return vb.init
}

// Delete deletes the GPU resources associated with this buffer
// just nils pointers -- not a big deal
func (vb *VectorsBuffer) Delete() {
	vb.Free()
}

// DeleteAllVectors deletes all Vectors defined for this buffer (calls Delete first)
func (vb *VectorsBuffer) DeleteAllVectors() {
	vb.Delete()
	vb.vecs = nil
	vb.stride = 0
	vb.offs = nil
	vb.nInter = 0
	vb.ln = 0
	vb.totLn = 0
}

// DeleteVectorsByName deletes Vectors of given name (calls Delete first)
func (vb *VectorsBuffer) DeleteVectorsByName(name string) {
	vb.Delete()
	for i, v := range vb.vecs {
		if v.name == name {
			vb.vecs = append(vb.vecs[:i], vb.vecs[i+1:]...)
			return
		}
	}
	log.Printf("glos.VectorsBuffer DeleteVectorsByName: name %v not found\n", name)
}

// DeleteVectorsByRole deletes Vectors of given role (calls Delete first)
func (vb *VectorsBuffer) DeleteVectorsByRole(role VectorRoles) {
	vb.Delete()
	for i, v := range vb.vecs {
		if v.role == role {
			vb.vecs = append(vb.vecs[:i], vb.vecs[i+1:]...)
			return
		}
	}
	log.Printf("glos.VectorsBuffer DeleteVectorsByRole: role %v not found\n", role)
}

// var glUsages = map[gpu.VectorUsages]uint32{
// 	gpu.StreamDraw:  gl.STREAM_DRAW,
// 	gpu.StreamRead:  gl.STREAM_READ,
// 	gpu.StreamCopy:  gl.STREAM_COPY,
// 	gpu.StaticDraw:  gl.STATIC_DRAW,
// 	gpu.StaticRead:  gl.STATIC_READ,
// 	gpu.StaticCopy:  gl.STATIC_COPY,
// 	gpu.DynamicDraw: gl.DYNAMIC_DRAW,
// 	gpu.DynamicRead: gl.DYNAMIC_READ,
// 	gpu.DynamicCopy: gl.DYNAMIC_COPY,
// }
