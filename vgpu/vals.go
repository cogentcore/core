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
	"cogentcore.org/core/math32"
	"cogentcore.org/core/vgpu/szalloc"
	vk "github.com/goki/vulkan"
)

// Value represents a specific value of a Var variable.
type Value struct {

	// name of this value, named by default as the variable name_idx
	Name string

	// index of this value within the Var list of values
	Index int

	// actual number of elements in an array -- 1 means scalar / singular value.  If 0, this is a dynamically sized item and the size must be set.
	N int

	// offset in bytes from start of memory buffer
	Offset int

	// val state flags
	Flags ValueFlags

	// if N > 1 (array) then this is the effective size of each element, which must be aligned to 16 byte modulo for Uniform types.  non naturally aligned types require slower element-by-element syncing operations, instead of memcopy.
	ElSize int

	// total memory size of this value in bytes, as allocated, including array alignment but not any additional buffer-required alignment padding
	AllocSize int

	// for Texture Var roles, this is the Texture
	Texture *Texture

	// pointer to the start of the staging memory for this value
	MemPtr unsafe.Pointer `display:"-"`
}

// HasFlag checks if flag is set
// using atomic, safe for concurrent access
func (vl *Value) HasFlag(flag ValueFlags) bool {
	return vl.Flags.HasFlag(flag)
}

// SetFlag sets flag(s) using atomic, safe for concurrent access
func (vl *Value) SetFlag(on bool, flag ...enums.BitFlag) {
	vl.Flags.SetFlag(on, flag...)
}

// IsMod returns true if the value has been modified since last memory sync
func (vl *Value) IsMod() bool {
	return vl.HasFlag(ValueMod)
}

// SetMod sets modified flag
func (vl *Value) SetMod() {
	vl.SetFlag(true, ValueMod)
}

// ClearMod clears modified flag
func (vl *Value) ClearMod() {
	vl.SetFlag(false, ValueMod)
}

// Init initializes value based on variable and index within list of vals for this var
func (vl *Value) Init(gp *GPU, vr *Var, idx int) {
	vl.Index = idx
	vl.Name = fmt.Sprintf("%s_%d", vr.Name, vl.Index)
	vl.N = vr.ArrayN
	if vr.Role >= TextureRole {
		vl.Texture = &Texture{}
		vl.Texture.GPU = gp
		vl.Texture.Defaults()
	}
}

// MemSize returns the memory allocation size for this value, in bytes
func (vl *Value) MemSize(vr *Var) int {
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
func (vl *Value) AllocHost(vr *Var, buff *MemBuff, buffPtr unsafe.Pointer, offset int) int {
	mem := vl.MemSize(vr)
	vl.AllocSize = mem
	if mem == 0 {
		return 0
	}
	vl.MemPtr = unsafe.Pointer(uintptr(buffPtr) + uintptr(offset))
	vl.Offset = offset
	if vl.Texture != nil {
		vl.Texture.ConfigValueHost(buff, buffPtr, offset)
	} else {
		if vl.N > 1 && vl.ElSize != vr.SizeOf {
			vl.SetFlag(true, ValuePaddedArray)
		} else {
			vl.SetFlag(false, ValuePaddedArray)
		}
	}
	return mem
}

// Free resets the MemPtr for this value
func (vl *Value) Free() {
	vl.Offset = 0
	vl.MemPtr = nil
	if vl.Texture != nil {
		vl.Texture.Destroy()
	}
}

// Bytes returns byte array of the Value data, including any additional
// alignment  -- can be written to directly.
// Be mindful of potential padding and alignment issues relative to
// go-based storage.
// Set Mod flag when changes have been made.
func (vl *Value) Bytes() []byte {
	return (*[ByteCopyMemoryLimit]byte)(vl.MemPtr)[:vl.AllocSize]
}

// Floats32 returns math32.ArrayF32 of the Value data -- can be written to directly.
// Only recommended for Vertex data.  Otherwise, be mindful of potential padding
// and alignment issues relative to go-based storage.
// Set Mod flag when changes have been made.
func (vl *Value) Floats32() math32.ArrayF32 {
	nf := vl.AllocSize / 4
	return math32.ArrayF32((*[ByteCopyMemoryLimit]float32)(vl.MemPtr)[:nf])
}

// UInts32 returns math32.ArrayU32 of the Value data -- can be written to directly.
// Only recommended for Vertex data.  Otherwise, be mindful of potential padding
// and alignment issues relative to go-based storage.
// Set Mod flag when changes have been made.
func (vl *Value) UInts32() math32.ArrayU32 {
	nf := vl.AllocSize / 4
	return math32.ArrayU32((*[ByteCopyMemoryLimit]uint32)(vl.MemPtr)[:nf])
}

// PaddedArrayCheck checks if this is an array with padding on the elements
// due to alignment issues.  If this is the case, then direct copying is not
// possible.
func (vl *Value) PaddedArrayCheck() error {
	if vl.HasFlag(ValuePaddedArray) {
		return fmt.Errorf("vgpu.Value PaddedArrayCheck: this array value has padding around elements not present in Go version -- cannot copy directly: %s", vl.Name)
	}
	return nil
}

// CopyFromBytes copies bytes from given source pointer into memory,
// and sets Mod flag.  Use this for struct data types.
func (vl *Value) CopyFromBytes(srcPtr unsafe.Pointer) {
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
func (vl *Value) CopyToBytes(srcPtr unsafe.Pointer) {
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
func (vl *Value) SetGoImage(img image.Image, layer int, flipY bool) error {
	if vl.HasFlag(ValueTextureOwns) {
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
	if vl.HasFlag(ValueTextureOwns) {
		vl.Texture.AllocTexture()
		// svimg, _ := vl.Texture.GoImage()
		// images.Save(svimg, fmt.Sprintf("dimg_%d.png", vl.Index))
	}
	return err
}

// MemReg returns the memory region for this value
func (vl *Value) MemReg(vr *Var) MemReg {
	bt := vr.Role.BuffType()
	mr := MemReg{Offset: vl.Offset, Size: vl.AllocSize, BuffType: bt}
	if bt == StorageBuff {
		mr.BuffIndex = vr.StorageBuff
	}
	return mr
}

//////////////////////////////////////////////////////////////////
// Values

// Values is a list container of Value values, accessed by index or name
type Values struct {

	// values in indexed order
	Values []*Value

	// map of vals by name -- only for specifically named vals vs. generically allocated ones -- names must be unique
	NameMap map[string]*Value

	// for texture values, this allocates textures to texture arrays by size -- used if On flag is set -- must call AllocTexBySize to allocate after ConfigGoImage is called on all vals.  Then call SetGoImage method on Values to set the Go Image for each val -- this automatically redirects to the group allocated images.
	TexSzAlloc szalloc.SzAlloc

	// for texture values, if AllocTexBySize is called, these are the actual allocated image arrays that hold the grouped images (size = TexSzAlloc.GpAllocs.
	GpTexValues []*Value
}

// ConfigValues configures given number of values in the list for given variable.
// If the same number of vals is given, nothing is done, so it is safe to call
// repeatedly.  Otherwise, any existing vals will be deleted -- the Memory system
// must free all associated memory prior!
// Returns true if new config made, else false if same size.
func (vs *Values) ConfigValues(gp *GPU, dev vk.Device, vr *Var, nvals int) bool {
	if len(vs.Values) == nvals {
		return false
	}
	vs.NameMap = make(map[string]*Value, nvals)
	vs.Values = make([]*Value, nvals)
	for i := 0; i < nvals; i++ {
		vl := &Value{}
		vl.Init(gp, vr, i)
		vs.Values[i] = vl
		if vr.TextureOwns {
			vl.SetFlag(true, ValueTextureOwns)
		}
		if vl.Texture != nil {
			vl.Texture.Dev = dev
		}
	}
	return true
}

// ValueByIndexTry returns Value at given index with range checking error message.
func (vs *Values) ValueByIndexTry(idx int) (*Value, error) {
	if idx >= len(vs.Values) || idx < 0 {
		err := fmt.Errorf("vgpu.Values:ValueByIndexTry index %d out of range", idx)
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vs.Values[idx], nil
}

// SetName sets name of given Value, by index, adds name to map, checking
// that it is not already there yet.  Returns val.
func (vs *Values) SetName(idx int, name string) (*Value, error) {
	vl, err := vs.ValueByIndexTry(idx)
	if err != nil {
		return nil, err
	}
	_, has := vs.NameMap[name]
	if has {
		err := fmt.Errorf("vgpu.Values:SetName name %s exists", name)
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	vl.Name = name
	vs.NameMap[name] = vl
	return vl, nil
}

// ValueByNameTry returns value by name, returning error if not found
func (vs *Values) ValueByNameTry(name string) (*Value, error) {
	vl, ok := vs.NameMap[name]
	if !ok {
		err := fmt.Errorf("vgpu.Values:ValueByNameTry name %s not found", name)
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vl, nil
}

//////////////////////////////////////////////////////////////////
// Values

// ActiveValues returns the Values to actually use for memory allocation etc
// this is Values list except for textures with TexSzAlloc.On active
func (vs *Values) ActiveValues() []*Value {
	if vs.TexSzAlloc.On && vs.GpTexValues != nil {
		return vs.GpTexValues
	}
	return vs.Values
}

// MemSize returns size across all Values in list
func (vs *Values) MemSize(vr *Var, alignBytes int) int {
	tsz := 0
	vals := vs.ActiveValues()
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
func (vs *Values) AllocHost(vr *Var, buff *MemBuff, offset int) int {
	tsz := 0
	vals := vs.ActiveValues()
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
func (vs *Values) Free() {
	vals := vs.ActiveValues()
	for _, vl := range vals {
		vl.Free()
	}
}

// Destroy frees all existing values and resets the list of Values so subsequent
// Config will start fresh (e.g., if Var type changes).
func (vs *Values) Destroy() {
	vs.Free()
	vs.Values = nil
	vs.GpTexValues = nil
	vs.TexSzAlloc.On = false
	vs.NameMap = nil
}

// ModRegs returns the regions of Values that have been modified
func (vs *Values) ModRegs(vr *Var) []MemReg {
	var mods []MemReg
	vals := vs.ActiveValues()
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
// This method creates actual image vals in GpTexValues, which
// are allocated.  Must call SetGoImage on Values here, which
// redirects to the proper allocated GpTexValues image and layer.
func (vs *Values) AllocTexBySize(gp *GPU, vr *Var) {
	if vr.TextureOwns {
		log.Println("vgpu.Values.AllocTexBySize: cannot use TextureOwns flag for this function.")
		vs.TexSzAlloc.On = false
		return
	}
	nv := len(vs.Values)
	if nv == 0 {
		vs.Free()
		vs.TexSzAlloc.On = false
		vs.GpTexValues = nil
		return
	}
	szs := make([]image.Point, nv)
	for i, vl := range vs.Values {
		szs[i] = vl.Texture.Format.Size
	}
	// 4,4 = MaxTexturesPerSet
	vs.TexSzAlloc.SetSizes(image.Point{4, 4}, MaxImageLayers, szs)
	vs.TexSzAlloc.Alloc()
	ng := len(vs.TexSzAlloc.GpAllocs)
	vs.GpTexValues = make([]*Value, ng)
	for i, sz := range vs.TexSzAlloc.GpSizes {
		nlay := len(vs.TexSzAlloc.GpAllocs[i])
		vl := &Value{}
		vl.Init(gp, vr, i)
		vs.GpTexValues[i] = vl
		vl.Texture.ConfigGoImage(sz, nlay)
	}
}

// SetGoImage calls SetGoImage on the proper Texture value for given index.
// if TexSzAlloc.On via AllocTexBySize then this is routed to the actual
// allocated image array, otherwise it goes directly to the standard Value.
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
func (vs *Values) SetGoImage(idx int, img image.Image, flipy bool) {
	if !vs.TexSzAlloc.On || vs.GpTexValues == nil {
		vl := vs.Values[idx]
		vl.SetGoImage(img, 0, flipy)
		return
	}
	idxs := vs.TexSzAlloc.ItemIndexes[idx]
	vl := vs.GpTexValues[idxs.GpIndex]
	vl.SetGoImage(img, idxs.ItemIndex, flipy)
}

////////////////////////////////////////////////////////////////
// Texture val functions

// AllocTextures allocates images on device memory
// only called on Role = TextureRole
func (vs *Values) AllocTextures(mm *Memory) {
	vals := vs.ActiveValues()
	for _, vl := range vals {
		if vl.Texture == nil {
			continue
		}
		vl.Texture.Dev = mm.Device.Device
		vl.Texture.AllocTexture()
	}
}

/////////////////////////////////////////////////////////////////////
// ValueFlags

// ValueFlags are bitflags for Value state
type ValueFlags int64 //enums:bitflag -trim-prefix Value

const (
	// ValueMod the value has been modified
	ValueMod ValueFlags = iota

	// ValuePaddedArray array had to be padded -- cannot access elements continuously
	ValuePaddedArray

	// ValueTextureOwns val owns and manages the host staging memory for texture.
	// based on Var TextureOwns -- for dynamically changing images.
	ValueTextureOwns
)
