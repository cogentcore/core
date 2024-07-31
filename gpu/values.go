// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"image"
	"log"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/slicesx"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// Value represents a specific value of a Var variable, with
// its own WebGPU Buffer or Texture associated with it.
// The Current active Value index can be set in the corresponding Var.Values.
// The Buffer for a Uniform or Storage value is created on the first
// SetValueFrom call, or
type Value struct {
	// name of this value, named by default as the variable name_idx
	Name string

	// index of this value within the Var list of values
	Index int

	// VarSize is the size of each Var element, which includes any fixed ArrayN
	// array size specified on the Var.
	VarSize int

	// DynamicN is the number of Var elements to encode within one
	// Value buffer, for Vertex values or DynamicOffset values
	// (otherwise it is always effectively 1).
	DynamicN int

	// AlignVarSize is VarSize subject to memory alignment constraints,
	// for DynamicN case.
	AlignVarSize int

	// AllocSize is total memory size of this value in bytes,
	// as allocated in the buffer.  For non-dynamic case, it is just VarSize.
	// For dynamic, it is DynamicN * AlignVarSize.
	AllocSize int

	role VarRoles

	// for this variable type, this is the alignment requirement in bytes,
	// for DynamicOffset variables.  This is 1 for Vertex buffer variables.
	alignBytes int

	// true if this is a dynamic variable (Vertex, DynamicOffset Uniform or Storage)
	isDynamic bool

	device Device

	// buffer for this value, makes it accessible to the GPU
	buffer *wgpu.Buffer `display:"-"`

	// for SampledTexture Var roles, this is the Texture.
	Texture *TextureSample

	TextureOwns bool
}

func NewValue(vr *Var, dev *Device, idx int) *Value {
	vl := &Value{}
	vl.init(vr, dev, idx)
	return vl
}

// MemSizeAlign returns the size aligned according to align byte increments
// e.g., if align = 16 and size = 12, it returns 16
func MemSizeAlign(size, align int) int {
	if size%align == 0 {
		return size
	}
	nb := size / align
	return (nb + 1) * align
}

// init initializes value based on variable and index
// within list of vals for this var.
func (vl *Value) init(vr *Var, dev *Device, idx int) {
	vl.role = vr.Role
	vl.device = *dev
	vl.Index = idx
	vl.Name = fmt.Sprintf("%s_%d", vr.Name, vl.Index)
	vl.VarSize = vr.MemSize()
	vl.alignBytes = vr.alignBytes
	vl.AlignVarSize = MemSizeAlign(vl.VarSize, vl.alignBytes)
	vl.isDynamic = vl.role == Vertex || vl.role == Index || vr.DynamicOffset
	vl.DynamicN = 1
	vl.TextureOwns = vr.TextureOwns
	if vr.Role >= SampledTexture {
		vl.Texture = NewTextureSample(dev)
	}
}

// MemSize returns the memory allocation size for this value, in bytes.
func (vl *Value) MemSize() int {
	if vl.Texture != nil {
		return vl.Texture.Format.TotalByteSize()
	}
	if vl.isDynamic {
		return vl.AlignVarSize * vl.DynamicN
	}
	return vl.VarSize
}

// CreateBuffer creates the GPU buffer for this value if it does not
// yet exist or is not the right size.
func (vl *Value) CreateBuffer() error {
	if vl.role == SampledTexture {
		return nil
	}
	sz := vl.MemSize()
	if sz == 0 {
		vl.Release()
		return nil
	}
	if sz == vl.AllocSize && vl.buffer != nil {
		return nil
	}
	vl.Release()
	buf, err := vl.device.Device.CreateBuffer(&wgpu.BufferDescriptor{
		Size:             uint64(sz),
		Label:            vl.Name,
		Usage:            vl.role.BufferUsages(),
		MappedAtCreation: false,
	})
	if errors.Log(err) != nil {
		return err
	}
	vl.AllocSize = sz
	vl.buffer = buf
	return nil
}

// Release releases the buffer / texture for this value
func (vl *Value) Release() {
	if vl.buffer != nil {
		vl.buffer.Release()
		vl.buffer = nil
	}
	if vl.Texture != nil {
		vl.Texture.Release()
		vl.Texture = nil
	}
}

// NilBufferCheckCheck checks if buffer is nil, returning error if so
func (vl *Value) NilBufferCheck() error {
	if vl.buffer == nil {
		return fmt.Errorf("gpu.Value NilBufferCheck: buffer is nil for value: %s", vl.Name)
	}
	return nil
}

// SetValueFrom copies given values into value buffer memory,
// making the buffer if it has not yet been constructed.
// IMPORTANT: do not use this for dynamic offset Uniform or
// Storage variables, as the alignment will not be correct;
// See SetDynamicFromBytes.
func SetValueFrom[E any](vl *Value, from []E) error {
	return vl.SetFromBytes(wgpu.ToBytes(from))
}

// SetDynamicValueFrom copies given values into value buffer memory,
// at the given dynamic variable index,
// making the buffer if it has not yet been constructed,
// for dynamic offset Uniform or Storage variables,
// which have alignment constraints.  Vertex variables
// should use standard SetValueFrom function.
// It is essential that DynamicN is set properly before
// calling this, and if that number has changed since
// last setting any values, _all_ of them must be set again.
func SetDynamicValueFrom[E any](vl *Value, idx int, from []E) error {
	return vl.SetDynamicFromBytes(idx, wgpu.ToBytes(from))
}

// SetFromBytes copies given bytes into value buffer memory,
// making the buffer if it has not yet been constructed.
// IMPORTANT: do not use this for dynamic offset Uniform or
// Storage variables, as the alignment will not be correct;
// See SetDynamicFromBytes.
func (vl *Value) SetFromBytes(from []byte) error {
	if vl.isDynamic && vl.alignBytes != 1 {
		err := fmt.Errorf("gpu.Value SetFromBytes %s: Cannot call this on a DynamicOffset Uniform or Storage variable; use SetDynamicValueFrom instead", vl.Name)
		return errors.Log(err)
	}
	nb := len(from)
	if vl.buffer == nil {
		buf, err := vl.device.Device.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    vl.Name,
			Contents: from,
			Usage:    vl.role.BufferUsages(),
		})
		if errors.Log(err) != nil {
			return err
		}
		vl.buffer = buf
		vl.AllocSize = nb
	} else {
		err := vl.device.Queue.WriteBuffer(vl.buffer, 0, from)
		if errors.Log(err) != nil {
			return err
		}
	}
	if vl.isDynamic {
		vl.DynamicN = nb / vl.AlignVarSize
		if nb%vl.AlignVarSize != 0 {
			err := fmt.Errorf("gpu.Value SetFromBytes %s, Size passed: %d is not an even multiple of the variable size: %d", vl.Name, nb, vl.AlignVarSize)
			return errors.Log(err)
		}
	}
	tb := vl.MemSize()
	if nb != tb {
		err := fmt.Errorf("gpu.Value SetFromBytes %s, Size passed: %d != Size expected %d", vl.Name, nb, tb)
		return errors.Log(err)
	}
	return nil
}

// SetDynamicFromBytes copies given values into value buffer memory,
// at the given dynamic variable index,
// making the buffer if it has not yet been constructed,
// for dynamic offset Uniform or Storage variables,
// which have alignment constraints.  Vertex variables
// should use standard SetValueFrom function.
// It is essential that DynamicN is set properly before
// calling this, and if that number has changed since
// last setting any values, _all_ of them must be set again.
func (vl *Value) SetDynamicFromBytes(idx int, from []byte) error {
	if !vl.isDynamic || vl.alignBytes == 1 {
		err := fmt.Errorf("gpu.Value SetDynamicFromBytes %s: Cannot call this on a non-DynamicOffset Uniform or Storage variable; use SetValueFrom instead", vl.Name)
		return errors.Log(err)
	}
	if idx >= vl.DynamicN {
		err := fmt.Errorf("gpu.Value SetDynamicFromBytes %s: Index: %d >= DynamicN: %d", vl.Name, idx, vl.DynamicN)
		return errors.Log(err)
	}
	if vl.buffer == nil {
		err := vl.CreateBuffer()
		if errors.Log(err) != nil {
			return err
		}
	}
	off := idx * vl.AlignVarSize
	err := vl.device.Queue.WriteBuffer(vl.buffer, uint64(off), from)
	if errors.Log(err) != nil {
		return err
	}
	return nil
}

// CopyValueToBytes copies given value buffer memory to given bytes,
// ensuring that the buffer is mapped and ready to be copied into.
// This automatically calls Unmap() after copying.
func CopyValueToBytes[E any](vl *Value, dest []E) error {
	return vl.CopyToBytes(wgpu.ToBytes(dest))
}

// CopyToBytes copies value buffer memory to given bytes,
// ensuring that the buffer is mapped and ready to be copied into.
// This automatically calls Unmap() after copying.
func (vl *Value) CopyToBytes(dest []byte) error {
	if err := vl.NilBufferCheck(); errors.Log(err) != nil {
		return err
	}
	var err error
	vl.buffer.MapAsync(wgpu.MapModeRead, 0, uint64(vl.AllocSize), func(stat wgpu.BufferMapAsyncStatus) {
		if stat != wgpu.BufferMapAsyncStatusSuccess {
			err = fmt.Errorf("gpu.Value CopyToBytesAsync %s: status is %s", vl.Name, stat.String())
			return
		}
		bm := vl.buffer.GetMappedRange(0, uint(vl.AllocSize))
		copy(dest, bm)
		vl.buffer.Unmap()
	})
	return err
}

func (vl *Value) bindGroupEntry(vr *Var) []wgpu.BindGroupEntry {
	if vr.Role >= SampledTexture {
		return []wgpu.BindGroupEntry{
			{
				Binding:     uint32(vr.Binding),
				TextureView: vl.Texture.view,
			},
			{
				Binding: uint32(vr.Binding + 1),
				Sampler: vl.Texture.Sampler.sampler,
			},
		}
	}
	vl.CreateBuffer() // ensure made
	return []wgpu.BindGroupEntry{{
		Binding: uint32(vr.Binding),
		Buffer:  vl.buffer,
		Size:    wgpu.WholeSize,
	},
	}
}

// SetFromGoImage sets Texture image data from an image.Image standard Go image,
// at given layer. This is most efficiently done using an image.RGBA, but other
// formats will be converted as necessary.
// If flipY is true then the Texture Y axis is flipped when copying into
// the image data.  Can avoid this by configuring texture coordinates to
// compensate.
// The Sampler is also configured at this point, with the current settings,
// so set those before making this call.
func (vl *Value) SetFromGoImage(img image.Image, layer int, flipY bool) *TextureSample {
	err := vl.Texture.SetFromGoImage(img, layer, flipY)
	errors.Log(err)
	err = vl.Texture.Sampler.Config(&vl.device)
	errors.Log(err)
	return vl.Texture
}

//////////////////////////////////////////////////////////////////
// Values

// Values is a list container of Value values, accessed by index or name.
type Values struct {
	// values in indexed order.
	Values []*Value

	// Current specifies the current value to use in rendering.
	Current int

	// DynamicIndex is the current index into a DynamicOffset variable
	// to use for the SetBindGroup call.  Note that this is an index,
	// not an offset, so it indexes the DynamicN Vars in the Value,
	// using the AlignVarSize to compute the dynamicOffset, which
	// is what is actually used.
	DynamicIndex int

	// map of vals by name, only for specifically named vals
	// vs. generically allocated ones. Names must be unique.
	NameMap map[string]*Value
}

// Add adds a new Value for given variable.
func (vs *Values) Add(vr *Var, dev *Device, name ...string) *Value {
	if len(name) == 1 && vs.NameMap == nil {
		vs.NameMap = make(map[string]*Value)
	}
	cn := len(vs.Values)
	vl := NewValue(vr, dev, cn)
	vs.Values = append(vs.Values, vl)
	if len(name) == 1 {
		vl.Name = name[0]
		vs.NameMap[vl.Name] = vl
	}
	return vl
}

// SetN sets specific number of values, returning true if changed.
func (vs *Values) SetN(vr *Var, dev *Device, nvals int) bool {
	cn := len(vs.Values)
	if cn == nvals {
		return false
	}
	vs.Values = slicesx.SetLength(vs.Values, nvals)
	for i := cn; i < nvals; i++ {
		vl := NewValue(vr, dev, cn)
		vs.Values[i] = vl
	}
	return true
}

// CurrentValue returns the current Value according to Current index.
func (vs *Values) CurrentValue() *Value {
	return vs.Values[vs.Current]
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
		err := fmt.Errorf("gpu.Values:SetName name %s exists", name)
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	vl.Name = name
	vs.NameMap[name] = vl
	return vl, nil
}

// ValueByIndexTry returns Value at given index with range checking error message.
func (vs *Values) ValueByIndexTry(idx int) (*Value, error) {
	if idx >= len(vs.Values) || idx < 0 {
		err := fmt.Errorf("gpu.Values:ValueByIndexTry index %d out of range", idx)
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vs.Values[idx], nil
}

// ValueByNameTry returns value by name, returning error if not found
func (vs *Values) ValueByNameTry(name string) (*Value, error) {
	vl, ok := vs.NameMap[name]
	if !ok {
		err := fmt.Errorf("gpu.Values:ValueByNameTry name %s not found", name)
		if Debug {
			log.Println(err)
		}
		return nil, err
	}
	return vl, nil
}

// Release frees all the value buffers / textures
func (vs *Values) Release() {
	for _, vl := range vs.Values {
		vl.Release()
	}
	vs.Values = nil
	vs.NameMap = nil
}

// MemSize returns size in bytes across all Values in list
func (vs *Values) MemSize() int {
	tsz := 0
	for _, vl := range vs.Values {
		tsz += vl.MemSize()
	}
	return tsz
}

// bindGroupEntry returns the BindGroupEntry for Current
// value for this variable.
func (vs *Values) bindGroupEntry(vr *Var) []wgpu.BindGroupEntry {
	vl := vs.CurrentValue()
	return vl.bindGroupEntry(vr)
}

func (vs *Values) dynamicOffset() uint32 {
	vl := vs.Values[0]
	return uint32(vl.AlignVarSize * vs.DynamicIndex)
}
