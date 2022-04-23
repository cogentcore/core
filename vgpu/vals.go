// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"unsafe"

	"github.com/goki/mat32"
)

// Val represents a specific value of a Var variable.
type Val struct {
	Name    string         `desc:"name of this value (not the name of the variable)"`
	Var     *Var           `desc:"variable that we are representing the value of"`
	N       int            `desc:"number of elements in an array -- 0 or 1 means scalar / singular value"`
	Offset  int            `desc:"offset in bytes from start of memory buffer"`
	MemPtr  unsafe.Pointer `desc:"pointer to the start of the staging memory for this value"`
	Mod     bool           `desc:"modified -- set when values are set"`
	Indexes string         `desc:"name of another Val to use for Indexes when accessing this vector data (e.g., as vertexes)"`
}

func (vl *Val) Init(name string, vr *Var, n int) {
	vl.Name = name
	vl.Var = vr
	vl.N = n
}

// MemSize returns total number of bytes of memory needed
func (vl *Val) MemSize() int {
	return vl.Var.SizeOf * vl.N
}

// Alloc allocates this value at given offset in owning Memory buffer.
// Computes the MemPtr for this item, and returns MemSize() of this
// value, so memory can increment to next item.
func (vl *Val) Alloc(buffPtr unsafe.Pointer, offset int) int {
	vl.MemPtr = unsafe.Pointer(uintptr(buffPtr) + uintptr(offset))
	return vl.MemSize()
}

// Free resets the MemPtr for this value
func (vl *Val) Free() {
	vl.Offset = 0
	vl.MemPtr = nil
}

// Bytes returns byte array of the Val data -- can be written to directly
// Set Mod flag when changes have been made.
func (vl *Val) Bytes() []byte {
	const m = 0x7fffffff
	return (*[m]byte)(vl.MemPtr)[:vl.MemSize()]
}

// Floats32 returns mat32.ArrayF32 of the Val data -- can be written to directly.
// Set Mod flag when changes have been made.
func (vl *Val) Floats32() mat32.ArrayF32 {
	nf := vl.MemSize() / 4
	const m = 0x7fffffff
	return mat32.ArrayF32((*[m]float32)(vl.MemPtr)[:nf])
}

// UInts32 returns mat32.ArrayU32 of the Val data -- can be written to directly.
// Set Mod flag when changes have been made.
func (vl *Val) UInts32() mat32.ArrayU32 {
	nf := vl.MemSize() / 4
	const m = 0x7fffffff
	return mat32.ArrayU32((*[m]uint32)(vl.MemPtr)[:nf])
}

// CopyBytes copies bytes from given source pointer into memory,
// and sets Mod flag.
func (vl *Val) CopyBytes(srcPtr unsafe.Pointer) {
	dst := vl.Bytes()
	const m = 0x7fffffff
	src := (*[m]byte)(srcPtr)[:vl.MemSize()]
	copy(dst, src)
	vl.Mod = true
}

//////////////////////////////////////////////////////////////////

// Vals is a container of Val values
type Vals struct {
	Vals    []*Val          `desc:"values in order added"`
	ValMap  map[string]*Val `desc:"map of all vals -- names must be unique"`
	TotSize int             `desc:"total size across all Vals -- computed during Alloc"`
}

// AddVal adds a new value
func (vs *Vals) AddVal(vr *Val) {
	if vs.ValMap == nil {
		vs.ValMap = make(map[string]*Val)
	}
	vs.Vals = append(vs.Vals, vr)
	vs.ValMap[vr.Name] = vr
}

// Add adds a new value
func (vs *Vals) Add(name string, vr *Var, n int) {
	vl := &Val{}
	vl.Init(name, vr, n)
	vs.AddVal(vl)
}

// MemSize returns size across all Vals
func (vs *Vals) MemSize() int {
	tsz := 0
	for _, vl := range vs.Vals {
		tsz += vl.MemSize()
	}
	return tsz
}

// Alloc allocates values at given offset in owning Memory buffer.
// Computes the MemPtr for this item, and returns TotSize
// across all vals.
func (vs *Vals) Alloc(buffPtr unsafe.Pointer, offset int) int {
	tsz := 0
	for _, vl := range vs.Vals {
		sz := vl.Alloc(buffPtr, offset)
		offset += sz
		tsz += sz
	}
	vs.TotSize = tsz
	return tsz
}

// Free resets the MemPtr for values
func (vs *Vals) Free() {
	for _, vl := range vs.Vals {
		vl.Free()
	}
	vs.TotSize = 0
}

// ModRegs returns the regions of Vals that have been modified
func (vs *Vals) ModRegs() []MemReg {
	var mods []MemReg
	for _, vl := range vs.Vals {
		if vl.Mod {
			mods = append(mods, MemReg{Offset: vl.Offset, Size: vl.MemSize()})
		}
	}
	return mods
}
