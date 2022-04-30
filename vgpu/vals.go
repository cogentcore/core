// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"
	"unsafe"

	"github.com/goki/mat32"
)

// Val represents a specific value of a Var variable.
type Val struct {
	Name    string         `desc:"name of this value (not the name of the variable)"`
	Var     *Var           `desc:"variable that we are representing the value of"`
	N       int            `desc:"number of elements in an array -- 0 or 1 means scalar / singular value"`
	Offset  int            `desc:"offset in bytes from start of memory buffer"`
	Indexes string         `desc:"name of another Val to use for Indexes when accessing this vector data (e.g., as vertexes)"`
	ElSize  int            `desc:"if N > 1 (array) then this is the effective size of each element, which must be aligned to 16 byte modulo for Uniform types.  non naturally-aligned types require slower element-by-element syncing operations, instead of memcopy."`
	MemSize int            `desc:"total memory size of this value, including array alignment but not any additional buffer-required alignment padding"`
	Image   *Image         `desc:"for Image Var roles, this is the Image"`
	Mod     bool           `inactive:"+" desc:"modified -- set when values are set"`
	MemPtr  unsafe.Pointer `view:"-" desc:"pointer to the start of the staging memory for this value"`
}

func (vl *Val) Init(name string, vr *Var, n int) {
	vl.Name = name
	vl.Var = vr
	vl.N = n
}

// BuffType returns the memory buffer type for this variable, based on Var.Role
func (vl *Val) BuffType() BuffTypes {
	return vl.Var.BuffType()
}

// AllocSize updates the memory allocation size -- called in Alloc
// returns MemSize.
func (vl *Val) AllocSize() int {
	if vl.N == 0 {
		vl.N = 1
	}
	if vl.N == 1 || vl.Var.Role < Uniform {
		vl.ElSize = vl.Var.SizeOf
		vl.MemSize = vl.ElSize * vl.N
		return vl.MemSize
	}
	vl.ElSize = MemSizeAlign(vl.Var.SizeOf, 16)
	vl.MemSize = vl.ElSize * vl.N
	return vl.MemSize
}

// Alloc allocates this value at given offset in owning Memory buffer.
// Computes the MemPtr for this item, and returns AllocSize() of this
// value, so memory can increment to next item.
// offsets are guaranteed to be properly aligned per minUniformBufferOffsetAlignment.
func (vl *Val) Alloc(buffPtr unsafe.Pointer, offset int) int {
	mem := vl.AllocSize()
	vl.MemPtr = unsafe.Pointer(uintptr(buffPtr) + uintptr(offset))
	vl.Offset = offset
	return mem
}

// Free resets the MemPtr for this value
func (vl *Val) Free() {
	vl.Offset = 0
	vl.MemPtr = nil
}

// Bytes returns byte array of the Val data, including any additional
// alignment  -- can be written to directly.
// Be mindful of potential padding and alignment issues relative to
// go-based storage.
// Set Mod flag when changes have been made.
func (vl *Val) Bytes() []byte {
	const m = 0x7fffffff
	return (*[m]byte)(vl.MemPtr)[:vl.MemSize]
}

// Floats32 returns mat32.ArrayF32 of the Val data -- can be written to directly.
// Only recommended for Vertex data.  Otherwise, be mindful of potential padding
// and alignment issues relative to go-based storage.
// Set Mod flag when changes have been made.
func (vl *Val) Floats32() mat32.ArrayF32 {
	nf := vl.MemSize / 4
	const m = 0x7fffffff
	return mat32.ArrayF32((*[m]float32)(vl.MemPtr)[:nf])
}

// UInts32 returns mat32.ArrayU32 of the Val data -- can be written to directly.
// Only recommended for Vertex data.  Otherwise, be mindful of potential padding
// and alignment issues relative to go-based storage.
// Set Mod flag when changes have been made.
func (vl *Val) UInts32() mat32.ArrayU32 {
	nf := vl.MemSize / 4
	const m = 0x7fffffff
	return mat32.ArrayU32((*[m]uint32)(vl.MemPtr)[:nf])
}

// PaddedArrayCheck checks if this is an array with padding on the elements
// due to alignment issues.  If this is the case, then direct copying is not
// possible.
func (vl *Val) PaddedArrayCheck() error {
	if vl.N > 1 && vl.Var.SizeOf != vl.ElSize {
		return fmt.Errorf("vgpu.Val PaddedArrayCheck: this array value has padding around elements not present in Go version -- cannot copy directly: %s", vl.Name)
	}
	return nil
}

// CopyBytes copies bytes from given source pointer into memory,
// and sets Mod flag.
func (vl *Val) CopyBytes(srcPtr unsafe.Pointer) {
	if err := vl.PaddedArrayCheck(); err != nil {
		log.Println(err)
		return
	}
	dst := vl.Bytes()
	const m = 0x7fffffff
	src := (*[m]byte)(srcPtr)[:vl.MemSize]
	copy(dst, src)
	vl.Mod = true
}

// MemReg returns the memory region for this value
func (vl *Val) MemReg() MemReg {
	return MemReg{Offset: vl.Offset, Size: vl.MemSize}
}

//////////////////////////////////////////////////////////////////

// Vals is a container of Val values
type Vals struct {
	Vals   []*Val          `desc:"values in order added"`
	ValMap map[string]*Val `desc:"map of all vals -- names must be unique"`
}

// AddVal adds given value
func (vs *Vals) AddVal(vr *Val) {
	if vs.ValMap == nil {
		vs.ValMap = make(map[string]*Val)
	}
	vs.Vals = append(vs.Vals, vr)
	vs.ValMap[vr.Name] = vr
}

// Add adds a new value for given variable, with given number of array elements
// 0 = no array
func (vs *Vals) Add(name string, vr *Var, n int) *Val {
	vl := &Val{}
	vl.Init(name, vr, n)
	vs.AddVal(vl)
	return vl
}

// ValByNameTry returns value by name, returning error if not found
func (vs *Vals) ValByNameTry(name string) (*Val, error) {
	vl, ok := vs.ValMap[name]
	if !ok {
		err := fmt.Errorf("Value named %s not found", name)
		return nil, err
	}
	return vl, nil
}

// MemSize returns size across all Vals
func (vs *Vals) MemSize(bt BuffTypes) int {
	tsz := 0
	for _, vl := range vs.Vals {
		if vl.BuffType() == bt {
			tsz += vl.AllocSize()
		}
	}
	return tsz
}

// Alloc allocates values at given offset in given Memory buffer.
// Computes the MemPtr for each item, and returns TotSize
// across all vals.  The effective offset increment (based on size) is
// aligned at the given align byte level, which should be
// MinUniformBufferOffsetAlignment from gpu.
func (vs *Vals) Alloc(buff *MemBuff, offset int) int {
	tsz := 0
	for _, vl := range vs.Vals {
		if vl.BuffType() != buff.Type {
			continue
		}
		sz := vl.Alloc(buff.HostPtr, offset)
		esz := MemSizeAlign(sz, buff.AlignBytes)
		offset += esz
		tsz += esz
	}
	return tsz
}

// Free resets the MemPtr for values
func (vs *Vals) Free(buff *MemBuff) {
	for _, vl := range vs.Vals {
		if vl.BuffType() != buff.Type {
			continue
		}
		vl.Free()
	}
}

// ModRegs returns the regions of Vals that have been modified
func (vs *Vals) ModRegs() []MemReg {
	var mods []MemReg
	for _, vl := range vs.Vals {
		if vl.Mod {
			mods = append(mods, vl.MemReg())
		}
	}
	return mods
}
