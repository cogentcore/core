// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"image"
	"log"
	"unsafe"

	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Val represents a specific value of a Var variable.
type Val struct {
	Name      string         `desc:"name of this value, named by default as the variable name_idx"`
	Idx       int            `desc:"index of this value within the Var list of values"`
	N         int            `desc:"actual number of elements in an array -- 1 means scalar / singular value.  If 0, this is a dynamically sized item and the size must be set."`
	Offset    int            `desc:"offset in bytes from start of memory buffer"`
	Flags     int32          `desc:"val state flags"`
	ElSize    int            `desc:"if N > 1 (array) then this is the effective size of each element, which must be aligned to 16 byte modulo for Uniform types.  non naturally-aligned types require slower element-by-element syncing operations, instead of memcopy."`
	AllocSize int            `desc:"total memory size of this value in bytes, as allocated, including array alignment but not any additional buffer-required alignment padding"`
	Texture   *Texture       `desc:"for Texture Var roles, this is the Texture"`
	MemPtr    unsafe.Pointer `view:"-" desc:"pointer to the start of the staging memory for this value"`
}

// HasFlag checks if flag is set
// using atomic, safe for concurrent access
func (vl *Val) HasFlag(flag ValFlags) bool {
	return bitflag.HasAtomic32(&vl.Flags, int(flag))
}

// SetFlag sets flag(s) using atomic, safe for concurrent access
func (vl *Val) SetFlag(flag ...int) {
	bitflag.SetAtomic32(&vl.Flags, flag...)
}

// ClearFlag clears flag(s) using atomic, safe for concurrent access
func (vl *Val) ClearFlag(flag ...int) {
	bitflag.ClearAtomic32(&vl.Flags, flag...)
}

// IsMod returns true if the value has been modified since last memory sync
func (vl *Val) IsMod() bool {
	return vl.HasFlag(ValMod)
}

// SetMod sets modified flag
func (vl *Val) SetMod() {
	vl.SetFlag(int(ValMod))
}

// ClearMod clears modified flag
func (vl *Val) ClearMod() {
	vl.ClearFlag(int(ValMod))
}

// Init initializes value based on variable and index within list of vals for this var
func (vl *Val) Init(vr *Var, idx int) {
	vl.Idx = idx
	vl.Name = fmt.Sprintf("%s_%d", vr.Name, vl.Idx)
	vl.N = vr.ArrayN
	if vr.Role >= TextureRole {
		vl.Texture = &Texture{}
		vl.Texture.Defaults()
	}
}

// MemSize returns the memory allocation size for this value, in bytes
func (vl *Val) MemSize(vr *Var) int {
	if vl.N == 0 {
		vl.N = 1
	}
	switch {
	case vr.Role >= TextureRole:
		if vr.TextureOwns {
			return 0
		} else {
			return vl.Texture.Format.ByteSize()
		}
	case vl.N == 1 || vr.Role < Uniform:
		vl.ElSize = vr.SizeOf
		return vl.ElSize * vl.N
	default:
		vl.ElSize = MemSizeAlign(vr.SizeOf, 16) // todo: test this!
		return vl.ElSize * vl.N
	}
}

// AllocHost allocates this value at given offset in owning Memory buffer.
// Computes the MemPtr for this item, and returns AllocSize() of this
// value, so memory can increment to next item.
// offsets are guaranteed to be properly aligned per minUniformBufferOffsetAlignment.
func (vl *Val) AllocHost(vr *Var, buff *MemBuff, buffPtr unsafe.Pointer, offset int) int {
	mem := vl.MemSize(vr)
	vl.AllocSize = mem
	if mem == 0 {
		return 0
	}
	vl.MemPtr = unsafe.Pointer(uintptr(buffPtr) + uintptr(offset))
	vl.Offset = offset
	if vl.Texture != nil {
		vl.Texture.ConfigValHost(buff, buffPtr, offset)
	} else {
		if vl.N > 1 && vl.ElSize != vr.SizeOf {
			vl.SetFlag(int(ValPaddedArray))
		} else {
			vl.ClearFlag(int(ValPaddedArray))
		}
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
	return (*[m]byte)(vl.MemPtr)[:vl.AllocSize]
}

// Floats32 returns mat32.ArrayF32 of the Val data -- can be written to directly.
// Only recommended for Vertex data.  Otherwise, be mindful of potential padding
// and alignment issues relative to go-based storage.
// Set Mod flag when changes have been made.
func (vl *Val) Floats32() mat32.ArrayF32 {
	nf := vl.AllocSize / 4
	const m = 0x7fffffff
	return mat32.ArrayF32((*[m]float32)(vl.MemPtr)[:nf])
}

// UInts32 returns mat32.ArrayU32 of the Val data -- can be written to directly.
// Only recommended for Vertex data.  Otherwise, be mindful of potential padding
// and alignment issues relative to go-based storage.
// Set Mod flag when changes have been made.
func (vl *Val) UInts32() mat32.ArrayU32 {
	nf := vl.AllocSize / 4
	const m = 0x7fffffff
	return mat32.ArrayU32((*[m]uint32)(vl.MemPtr)[:nf])
}

// PaddedArrayCheck checks if this is an array with padding on the elements
// due to alignment issues.  If this is the case, then direct copying is not
// possible.
func (vl *Val) PaddedArrayCheck() error {
	if vl.HasFlag(ValPaddedArray) {
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
	src := (*[m]byte)(srcPtr)[:vl.AllocSize]
	copy(dst, src)
	vl.SetMod()
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
	if vl.HasFlag(ValTextureOwns) {
		vl.Texture.ConfigGoImage(img)
		vl.Texture.AllocHost()
	}
	err := vl.Texture.SetGoImage(img, flipY)
	if err != nil {
		fmt.Println(err)
	} else {
		vl.SetMod()
	}
	if vl.HasFlag(ValTextureOwns) {
		vl.Texture.AllocTexture()
	}
	return err
}

// MemReg returns the memory region for this value
func (vl *Val) MemReg() MemReg {
	return MemReg{Offset: vl.Offset, Size: vl.AllocSize}
}

//////////////////////////////////////////////////////////////////
// Vals

// Vals is a list container of Val values, accessed by index or name
type Vals struct {
	Vals    []*Val          `desc:"values in indexed order"`
	NameMap map[string]*Val `desc:"map of vals by name -- only for specifically named vals vs. generically allocated ones -- names must be unique"`
}

// ConfigVals configures given number of values in the list for given variable.
// Any existing vals will be deleted -- must free all associated memory prior!
func (vs *Vals) ConfigVals(vr *Var, nvals int) {
	vs.NameMap = make(map[string]*Val, nvals)
	vs.Vals = make([]*Val, nvals)
	for i := 0; i < nvals; i++ {
		vl := &Val{}
		vl.Init(vr, i)
		vs.Vals[i] = vl
		if vr.TextureOwns {
			vl.SetFlag(int(ValTextureOwns))
		}
	}
}

// ValByIdxTry returns Val at given index with range checking error message.
func (vs *Vals) ValByIdxTry(idx int) (*Val, error) {
	if idx >= len(vs.Vals) || idx < 0 {
		err := fmt.Errorf("vgpu.Vals:ValByIdxTry index %d out of range", idx)
		if TheGPU.Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vs.Vals[idx], nil
}

// SetName sets name of given Val, by index, adds name to map, checking
// that it is not already there yet.  Returns val.
func (vs *Vals) SetName(idx int, name string) (*Val, error) {
	vl, err := vs.ValByIdxTry(idx)
	if err != nil {
		return nil, err
	}
	_, has := vs.NameMap[name]
	if has {
		err := fmt.Errorf("vgpu.Vals:SetName name %s exists", name)
		if TheGPU.Debug {
			log.Println(err)
		}
		return nil, err
	}
	vl.Name = name
	vs.NameMap[name] = vl
	return vl, nil
}

// ValByNameTry returns value by name, returning error if not found
func (vs *Vals) ValByNameTry(name string) (*Val, error) {
	vl, ok := vs.NameMap[name]
	if !ok {
		err := fmt.Errorf("vgpu.Vals:ValByNameTry name %s not found", name)
		if TheGPU.Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vl, nil
}

//////////////////////////////////////////////////////////////////
// Vals

// MemSize returns size across all Vals in list
func (vs *Vals) MemSize(vr *Var, alignBytes int) int {
	offset := 0
	tsz := 0
	for _, vl := range vs.Vals {
		sz := vl.MemSize(vr)
		if sz == 0 {
			continue
		}
		esz := MemSizeAlign(sz, alignBytes)
		offset += esz
		tsz += esz
	}
	return tsz
}

// AllocHost allocates values at given offset in given Memory buffer.
// Computes the MemPtr for each item, and returns TotSize
// across all vals.  The effective offset increment (based on size) is
// aligned at the given align byte level, which should be
// MinUniformBufferOffsetAlignment from gpu.
func (vs *Vals) AllocHost(vr *Var, buff *MemBuff, offset int) int {
	tsz := 0
	for _, vl := range vs.Vals {
		sz := vl.AllocHost(vr, buff, buff.HostPtr, offset)
		if sz == 0 {
			continue
		}
		esz := MemSizeAlign(sz, buff.AlignBytes)
		offset += esz
		tsz += esz
	}
	return tsz
}

// Free resets the MemPtr for values, resets any self-owned resources (Textures)
func (vs *Vals) Free() {
	for _, vl := range vs.Vals {
		vl.Free()
	}
}

// ModRegs returns the regions of Vals that have been modified
func (vs *Vals) ModRegs() []MemReg {
	var mods []MemReg
	for _, vl := range vs.Vals {
		if vl.IsMod() {
			mods = append(mods, vl.MemReg())
			vl.ClearMod() // assuming it will clear now..
		}
	}
	return mods
}

////////////////////////////////////////////////////////////////
// Texture val functions

// AllocTextures allocates images on device memory
// only called on Role = TextureRole
func (vs *Vals) AllocTextures(mm *Memory) {
	for _, vl := range vs.Vals {
		if vl.Texture == nil {
			continue
		}
		vl.Texture.Dev = mm.Device.Device
		vl.Texture.AllocTexture()
	}
}

/////////////////////////////////////////////////////////////////////
// ValFlags

// ValFlags are bitflags for Val state
type ValFlags int32

const (
	// ValMod the value has been modified
	ValMod ValFlags = iota

	// ValPaddedArray array had to be padded -- cannot access elements continuously
	ValPaddedArray

	// ValTextureOwns val owns and manages the host staging memory for texture.
	// based on Var TextureOwns -- for dynamically changings images.
	ValTextureOwns

	ValFlagsN
)

//go:generate stringer -type=ValFlags

var KiT_ValFlags = kit.Enums.AddEnum(ValFlagsN, kit.BitFlag, nil)
