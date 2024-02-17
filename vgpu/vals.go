// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"image"
	"log"
	"unsafe"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/vgpu/szalloc"
	vk "github.com/goki/vulkan"
)

// Val represents a specific value of a Var variable.
type Val struct {

	// name of this value, named by default as the variable name_idx
	Name string

	// index of this value within the Var list of values
	Idx int

	// actual number of elements in an array -- 1 means scalar / singular value.  If 0, this is a dynamically sized item and the size must be set.
	N int

	// offset in bytes from start of memory buffer
	Offset int

	// val state flags
	Flags ValFlags

	// if N > 1 (array) then this is the effective size of each element, which must be aligned to 16 byte modulo for Uniform types.  non naturally-aligned types require slower element-by-element syncing operations, instead of memcopy.
	ElSize int

	// total memory size of this value in bytes, as allocated, including array alignment but not any additional buffer-required alignment padding
	AllocSize int

	// for Texture Var roles, this is the Texture
	Texture *Texture

	// pointer to the start of the staging memory for this value
	MemPtr unsafe.Pointer `view:"-"`
}

// HasFlag checks if flag is set
// using atomic, safe for concurrent access
func (vl *Val) HasFlag(flag ValFlags) bool {
	return vl.Flags.HasFlag(flag)
}

// SetFlag sets flag(s) using atomic, safe for concurrent access
func (vl *Val) SetFlag(on bool, flag ...enums.BitFlag) {
	vl.Flags.SetFlag(on, flag...)
}

// IsMod returns true if the value has been modified since last memory sync
func (vl *Val) IsMod() bool {
	return vl.HasFlag(ValMod)
}

// SetMod sets modified flag
func (vl *Val) SetMod() {
	vl.SetFlag(true, ValMod)
}

// ClearMod clears modified flag
func (vl *Val) ClearMod() {
	vl.SetFlag(false, ValMod)
}

// Init initializes value based on variable and index within list of vals for this var
func (vl *Val) Init(gp *GPU, vr *Var, idx int) {
	vl.Idx = idx
	vl.Name = fmt.Sprintf("%s_%d", vr.Name, vl.Idx)
	vl.N = vr.ArrayN
	if vr.Role >= TextureRole {
		vl.Texture = &Texture{}
		vl.Texture.GPU = gp
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
			return vl.Texture.Format.TotalByteSize()
		}
	case vl.N == 1 || vr.Role < Uniform:
		vl.ElSize = vr.SizeOf
		return vl.ElSize * vl.N
	case vr.Role == Uniform:
		vl.ElSize = MemSizeAlign(vr.SizeOf, 16) // todo: test this!
		return vl.ElSize * vl.N
	default: // storage is ok with anything?
		// vl.ElSize = MemSizeAlign(vr.SizeOf, 16) // todo: test this!
		vl.ElSize = vr.SizeOf
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
			vl.SetFlag(true, ValPaddedArray)
		} else {
			vl.SetFlag(false, ValPaddedArray)
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
	return (*[ByteCopyMemoryLimit]byte)(vl.MemPtr)[:vl.AllocSize]
}

// Floats32 returns mat32.ArrayF32 of the Val data -- can be written to directly.
// Only recommended for Vertex data.  Otherwise, be mindful of potential padding
// and alignment issues relative to go-based storage.
// Set Mod flag when changes have been made.
func (vl *Val) Floats32() mat32.ArrayF32 {
	nf := vl.AllocSize / 4
	return (*[ByteCopyMemoryLimit]float32)(vl.MemPtr)[:nf]
}

// UInts32 returns mat32.ArrayU32 of the Val data -- can be written to directly.
// Only recommended for Vertex data.  Otherwise, be mindful of potential padding
// and alignment issues relative to go-based storage.
// Set Mod flag when changes have been made.
func (vl *Val) UInts32() mat32.ArrayU32 {
	nf := vl.AllocSize / 4
	return (*[ByteCopyMemoryLimit]uint32)(vl.MemPtr)[:nf]
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

// CopyFromBytes copies bytes from given source pointer into memory,
// and sets Mod flag.  Use this for struct data types.
func (vl *Val) CopyFromBytes(srcPtr unsafe.Pointer) {
	if err := vl.PaddedArrayCheck(); err != nil {
		log.Println(err)
		// return
	}
	dst := vl.Bytes()
	src := (*[ByteCopyMemoryLimit]byte)(srcPtr)[:vl.AllocSize]
	copy(dst, src)
	vl.SetMod()
}

// CopyToBytes copies bytes from val to given source pointer into memory.
// Use this for struct data types to retrieve computed results.
func (vl *Val) CopyToBytes(srcPtr unsafe.Pointer) {
	if err := vl.PaddedArrayCheck(); err != nil {
		log.Println(err)
		return
	}
	dst := vl.Bytes()
	src := (*[ByteCopyMemoryLimit]byte)(srcPtr)[:vl.AllocSize]
	copy(src, dst)
}

// SetGoImage sets Texture image data from an *image.RGBA standard Go image,
// at given layer, and sets the Mod flag, so it will be sync'd by Memory
// or if TextureOwns is set for the var, it allocates Host memory.
// This is most efficiently done using an image.RGBA, but other
// formats will be converted as necessary.
// If flipY is true then the Image Y axis is flipped when copying into
// the image data (requires row-by-row copy) -- can avoid this
// by configuring texture coordinates to compensate.
func (vl *Val) SetGoImage(img image.Image, layer int, flipY bool) error {
	if vl.HasFlag(ValTextureOwns) {
		if layer == 0 && vl.Texture.Format.Layers <= 1 {
			vl.Texture.ConfigGoImage(img.Bounds().Size(), layer+1)
		}
		vl.Texture.AllocHost()
	}
	err := vl.Texture.SetGoImage(img, layer, flipY)
	if err != nil {
		fmt.Println(err)
	} else {
		vl.SetMod()
	}
	if vl.HasFlag(ValTextureOwns) {
		vl.Texture.AllocTexture()
		// svimg, _ := vl.Texture.GoImage()
		// images.Save(svimg, fmt.Sprintf("dimg_%d.png", vl.Idx))
	}
	return err
}

// MemReg returns the memory region for this value
func (vl *Val) MemReg(vr *Var) MemReg {
	bt := vr.Role.BuffType()
	mr := MemReg{Offset: vl.Offset, Size: vl.AllocSize, BuffType: bt}
	if bt == StorageBuff {
		mr.BuffIdx = vr.StorageBuff
	}
	return mr
}

//////////////////////////////////////////////////////////////////
// Vals

// Vals is a list container of Val values, accessed by index or name
type Vals struct {

	// values in indexed order
	Vals []*Val

	// map of vals by name -- only for specifically named vals vs. generically allocated ones -- names must be unique
	NameMap map[string]*Val

	// for texture values, this allocates textures to texture arrays by size -- used if On flag is set -- must call AllocTexBySize to allocate after ConfigGoImage is called on all vals.  Then call SetGoImage method on Vals to set the Go Image for each val -- this automatically redirects to the group allocated images.
	TexSzAlloc szalloc.SzAlloc

	// for texture values, if AllocTexBySize is called, these are the actual allocated image arrays that hold the grouped images (size = TexSzAlloc.GpAllocs.
	GpTexVals []*Val
}

// ConfigVals configures given number of values in the list for given variable.
// If the same number of vals is given, nothing is done, so it is safe to call
// repeatedly.  Otherwise, any existing vals will be deleted -- the Memory system
// must free all associated memory prior!
// Returns true if new config made, else false if same size.
func (vs *Vals) ConfigVals(gp *GPU, dev vk.Device, vr *Var, nvals int) bool {
	if len(vs.Vals) == nvals {
		return false
	}
	vs.NameMap = make(map[string]*Val, nvals)
	vs.Vals = make([]*Val, nvals)
	for i := 0; i < nvals; i++ {
		vl := &Val{}
		vl.Init(gp, vr, i)
		vs.Vals[i] = vl
		if vr.TextureOwns {
			vl.SetFlag(true, ValTextureOwns)
		}
		if vl.Texture != nil {
			vl.Texture.Dev = dev
		}
	}
	return true
}

// ValByIdxTry returns Val at given index with range checking error message.
func (vs *Vals) ValByIdxTry(index int) (*Val, error) {
	if index >= len(vs.Vals) || index < 0 {
		err := fmt.Errorf("vgpu.Vals:ValByIdxTry index %d out of range", index)
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vs.Vals[index], nil
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
		if Debug {
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
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vl, nil
}

//////////////////////////////////////////////////////////////////
// Vals

// ActiveVals returns the Vals to actually use for memory allocation etc
// this is Vals list except for textures with TexSzAlloc.On active
func (vs *Vals) ActiveVals() []*Val {
	if vs.TexSzAlloc.On && vs.GpTexVals != nil {
		return vs.GpTexVals
	}
	return vs.Vals
}

// MemSize returns size across all Vals in list
func (vs *Vals) MemSize(vr *Var, alignBytes int) int {
	tsz := 0
	vals := vs.ActiveVals()
	for _, vl := range vals {
		sz := vl.MemSize(vr)
		if sz == 0 {
			continue
		}
		esz := MemSizeAlign(sz, alignBytes)
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
	vals := vs.ActiveVals()
	for _, vl := range vals {
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
	vals := vs.ActiveVals()
	for _, vl := range vals {
		vl.Free()
	}
}

// Destroy frees all existing values and resets the list of Vals so subsequent
// Config will start fresh (e.g., if Var type changes).
func (vs *Vals) Destroy() {
	vs.Free()
	vs.Vals = nil
	vs.GpTexVals = nil
	vs.TexSzAlloc.On = false
	vs.NameMap = nil
}

// ModRegs returns the regions of Vals that have been modified
func (vs *Vals) ModRegs(vr *Var) []MemReg {
	var mods []MemReg
	vals := vs.ActiveVals()
	for _, vl := range vals {
		if vl.IsMod() {
			mods = append(mods, vl.MemReg(vr))
			vl.ClearMod() // assuming it will clear now..
		}
	}
	return mods
}

// AllocTexBySize allocates textures by size so they fit within the
// MaxTexturesPerGroup.  Must call ConfigGoImage on the original
// values to set the sizes prior to calling this, and cannot have
// the TextureOwns flag set.  Also does not support arrays in source vals.
// Apps can always use szalloc.SzAlloc upstream of this to allocate.
// This method creates actual image vals in GpTexVals, which
// are allocated.  Must call SetGoImage on Vals here, which
// redirects to the proper allocated GpTexVals image and layer.
func (vs *Vals) AllocTexBySize(gp *GPU, vr *Var) {
	if vr.TextureOwns {
		log.Println("vgpu.Vals.AllocTexBySize: cannot use TextureOwns flag for this function.")
		vs.TexSzAlloc.On = false
		return
	}
	nv := len(vs.Vals)
	if nv == 0 {
		vs.Free()
		vs.TexSzAlloc.On = false
		vs.GpTexVals = nil
		return
	}
	szs := make([]image.Point, nv)
	for i, vl := range vs.Vals {
		szs[i] = vl.Texture.Format.Size
	}
	// 4,4 = MaxTexturesPerSet
	vs.TexSzAlloc.SetSizes(image.Point{X: 4, Y: 4}, MaxImageLayers, szs)
	vs.TexSzAlloc.Alloc()
	ng := len(vs.TexSzAlloc.GpAllocs)
	vs.GpTexVals = make([]*Val, ng)
	for i, sz := range vs.TexSzAlloc.GpSizes {
		nlay := len(vs.TexSzAlloc.GpAllocs[i])
		vl := &Val{}
		vl.Init(gp, vr, i)
		vs.GpTexVals[i] = vl
		vl.Texture.ConfigGoImage(sz, nlay)
	}
}

// SetGoImage calls SetGoImage on the proper Texture value for given index.
// if TexSzAlloc.On via AllocTexBySize then this is routed to the actual
// allocated image array, otherwise it goes directly to the standard Val.
//
// SetGoImage sets staging image data from a standard Go image at given layer.
// This is most efficiently done using an image.RGBA, but other
// formats will be converted as necessary.
// If flipY is true then the Image Y axis is flipped
// when copying into the image data, so that images will appear
// upright in the standard OpenGL Y-is-up coordinate system.
// If using the Y-is-down Vulkan coordinate system, don't flip.
// Only works if IsHostActive and Image Format is default vk.FormatR8g8b8a8Srgb,
// Must still call AllocImage to have image allocated on the device,
// and copy from this host staging data to the device.
func (vs *Vals) SetGoImage(idx int, img image.Image, flipy bool) {
	if !vs.TexSzAlloc.On || vs.GpTexVals == nil {
		vl := vs.Vals[idx]
		vl.SetGoImage(img, 0, flipy)
		return
	}
	idxs := vs.TexSzAlloc.ItemIdxs[idx]
	vl := vs.GpTexVals[idxs.GpIdx]
	vl.SetGoImage(img, idxs.ItemIdx, flipy)
}

////////////////////////////////////////////////////////////////
// Texture val functions

// AllocTextures allocates images on device memory
// only called on Role = TextureRole
func (vs *Vals) AllocTextures(mm *Memory) {
	vals := vs.ActiveVals()
	for _, vl := range vals {
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
type ValFlags int64 //enums:bitflag -trim-prefix Val

const (
	// ValMod the value has been modified
	ValMod ValFlags = iota

	// ValPaddedArray array had to be padded -- cannot access elements continuously
	ValPaddedArray

	// ValTextureOwns val owns and manages the host staging memory for texture.
	// based on Var TextureOwns -- for dynamically changings images.
	ValTextureOwns
)
