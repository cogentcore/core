// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"image"
	"log"
	"unsafe"

	"github.com/goki/mat32"
)

// Val represents a specific value of a Var variable.
type Val struct {
	Name    string         `desc:"name of this value, named by default as the variable name_idx"`
	N       int            `desc:"actual number of elements in an array -- 1 means scalar / singular value.  If 0, this is a dynamically sized item and the size must be set."`
	Offset  int            `desc:"offset in bytes from start of memory buffer"`
	Indexes string         `desc:"name of another Val to use for Indexes when accessing this vector data (e.g., as vertexes)"`
	ElSize  int            `desc:"if N > 1 (array) then this is the effective size of each element, which must be aligned to 16 byte modulo for Uniform types.  non naturally-aligned types require slower element-by-element syncing operations, instead of memcopy."`
	MemSize int            `desc:"total memory size of this value, including array alignment but not any additional buffer-required alignment padding"`
	Texture *Texture       `desc:"for Texture Var roles, this is the Texture"`
	Mod     bool           `inactive:"+" desc:"modified -- set when values are set"`
	Buff    *MemBuff       `desc:"memory buffer that manages our memory"`
	MemPtr  unsafe.Pointer `view:"-" desc:"pointer to the start of the staging memory for this value"`
}

// Init initializes value based on variable and index within list of vals for this var
func (vl *Val) Init(vr *Var, idx int) {
	vl.Name = fmt.Sprintf("%s_%d", vr.Name, idx)
	vl.N = vr.ArrayN
	if vr.Role >= TextureRole {
		vl.Texture = &Texture{}
		vl.Name = name
		vl.Texture.Defaults()
	}
}

// AllocSize updates the memory allocation size -- called in Alloc
// returns MemSize.
func (vl *Val) AllocSize() int {
	if vl.N == 0 {
		vl.N = 1
	}
	if vl.Var.Role >= TextureRole {
		if vl.Var.TextureOwns {
			vl.MemSize = 0
		} else {
			vl.MemSize = vl.Texture.Format.ByteSize()
		}
		return vl.MemSize
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
func (vl *Val) Alloc(buff *MemBuff, buffPtr unsafe.Pointer, offset int) int {
	mem := vl.AllocSize()
	if mem == 0 {
		return 0
	}
	vl.MemPtr = unsafe.Pointer(uintptr(buffPtr) + uintptr(offset))
	vl.Offset = offset
	if vl.Texture != nil {
		vl.Texture.ConfigValHost(buff, buffPtr, offset)
	}
	return mem
}

// Free resets the MemPtr for this value
func (vl *Val) Free() {
	vl.Offset = 0
	vl.MemPtr = nil
	if vl.Texture != nil {
		vl.Texture.Destroy()
	}
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

// SetGoImage sets Texture image data from an *image.RGBA standard Go image,
// and sets the Mod flag, so it will be sync'd up when memory is sync'd,
// or if TextureOwns is set for the var, it allocates Host memory.
// This is most efficiently done using an image.RGBA, but other
// formats will be converted as necessary.
// If flipY is true (default) then the Image Y axis is flipped
// when copying into the image data, so that images will appear
// upright in the standard OpenGL Y-is-Up coordinate system.
// If using the Y-is-down Vulkan coordinate system, don't flip.
func (vl *Val) SetGoImage(img image.Image, flipY bool) error {
	if vl.Var.TextureOwns {
		vl.Texture.ConfigGoImage(img)
		vl.Texture.AllocHost()
	}
	err := vl.Texture.SetGoImage(img, flipY)
	if err != nil {
		fmt.Println(err)
	} else {
		vl.Mod = true
	}
	if vl.Var.TextureOwns {
		vl.Texture.AllocTexture()
	}
	return err
}

// MemReg returns the memory region for this value
func (vl *Val) MemReg() MemReg {
	return MemReg{Offset: vl.Offset, Size: vl.MemSize}
}

//////////////////////////////////////////////////////////////////
// ValList

// ValList is a list container of Val values, accessed by index or name
type ValList struct {
	Vals   []*Val          `desc:"values in indexed order"`
	ValMap map[string]*Val `desc:"map of vals by name -- only for specifically named vals vs. generically allocated ones -- names must be unique"`
}

// ConfigVals configures given number of values in the list for given variable.
// Any existing vals will be deleted -- must free all associated memory prior!
func (vs *ValList) ConfigVals(vr *Val, nvals int) {
	vs.ValMap = make(map[string]*Val, nvals)
	vs.Vals = make([]*Val, nvals)
	for i := 0; i < nvals; i++ {
		vl := &Val{}
		vl.Init(vr, i)
		vs.Vals[i] = vl
	}
}

// ValByIdxTry returns Val at given index with range checking error message.
func (vs *ValList) ValByIdxTry(idx int) (*Val, error) {
	if idx >= len(vs.Vals) || idx < 0 {
		err := fmt.Errorf("vgpu.ValList:ValByIdxTry index %d out of range", idx)
		if TheGPU.Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vs.Vals[idx], nil
}

// SetName sets name of given Val, by index, adds name to map, checking
// that it is not already there yet.  Returns val.
func (vs *ValList) SetName(idx int, name string) (*Val, error) {
	vl, err := vs.ValByIdxTry(idx)
	if err != nil {
		return nil, err
	}
	nm, has := vs.ValMap[name]
	if has {
		err := fmt.Errorf("vgpu.ValList:SetName name %s exists", name)
		if TheGPU.Debug {
			log.Println(err)
		}
		return nil, err
	}
	vl.Name = name
	vl.ValMap[name] = vl
	return vl, nil
}

// ValByNameTry returns value by name, returning error if not found
func (vs *ValList) ValByNameTry(name string) (*Val, error) {
	vl, ok := vs.ValMap[name]
	if !ok {
		err := fmt.Errorf("vgpu.ValList:ValByNameTry name %s not found", name)
		if TheGPU.Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vl, nil
}

//////////////////////////////////////////////////////////////////
// ValList

// MemSize returns size across all Vals
func (vs *Vals) MemSize(buff *MemBuff) int {
	offset := 0
	tsz := 0
	for _, vl := range vs.Vals {
		if vl.BuffType() != buff.Type {
			continue
		}
		sz := vl.AllocSize()
		if sz == 0 {
			continue
		}
		esz := MemSizeAlign(sz, buff.AlignBytes)
		offset += esz
		tsz += esz
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
		sz := vl.Alloc(buff, buff.HostPtr, offset)
		if sz == 0 {
			continue
		}
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

// ModRegs returns the regions of Vals that have been modified in given
// type of buffer
func (vs *Vals) ModRegs(bt BuffTypes) []MemReg {
	var mods []MemReg
	for _, vl := range vs.Vals {
		if vl.Mod && vl.BuffType() == bt {
			mods = append(mods, vl.MemReg())
		}
	}
	return mods
}

////////////////////////////////////////////////////////////////
// Texture val functions

// AllocTextures allocates images on device memory
func (vs *Vals) AllocTextures(mm *Memory) {
	for _, vl := range vs.Vals {
		if vl.BuffType() != ImageBuff || vl.Texture == nil {
			continue
		}
		vl.Texture.Dev = mm.Device.Device
		vl.Texture.AllocTexture()
	}
}
