// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"image"
	"log"
	"log/slog"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/slicesx"
	"github.com/cogentcore/webgpu/wgpu"
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

	// DynamicIndex is the current index into a DynamicOffset variable
	// to use for the SetBindGroup call.  Note that this is an index,
	// not an offset, so it indexes the DynamicN Vars in the Value,
	// using the AlignVarSize to compute the dynamicOffset, which
	// is what is actually used.
	DynamicIndex int

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

	// dynamicBuffer is a CPU-based staging buffer for dynamic values
	// so you can separately set individual dynamic index values and
	// then efficiently copy the entire thing to the device buffer
	// once everything has been set.
	dynamicBuffer []byte

	// for SampledTexture Var roles, this is the Texture.
	// Can set Sampler parameters directly on this.
	Texture *TextureSample
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
	if vl.isDynamic { // Vertex, Index at this point
		vl.DynamicN = nb / vl.VarSize
	}
	tb := vl.MemSize()
	if nb != tb {
		err := fmt.Errorf("gpu.Value SetFromBytes %s, Size passed: %d != Size expected %d", vl.Name, nb, tb)
		return errors.Log(err)
	}
	if vl.buffer == nil || vl.AllocSize != tb {
		vl.Release()
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
	return nil
}

// SetDynamicValueFrom copies given values into a staging buffer
// at the given dynamic variable index, for dynamic offset
// Uniform or Storage variables, which have alignment constraints.
// Must call WriteDynamicBuffer after all such values have been updated,
// to actually copy the entire staging buffer data to the GPU device.
// Vertex variables must have separate values for each, and do not
// support dynamic indexing.
// It is essential that DynamicN is set properly before
// calling this.  Existing values will be preserved with
// changes in DynamicN to the extent possible.
func SetDynamicValueFrom[E any](vl *Value, idx int, from []E) error {
	return vl.SetDynamicFromBytes(idx, wgpu.ToBytes(from))
}

// SetDynamicFromBytes copies given values into a staging buffer
// at the given dynamic variable index, for dynamic offset
// Uniform or Storage variables, which have alignment constraints.
// See [SetDynamicValueFrom], which should generally be used,
// for further info.
func (vl *Value) SetDynamicFromBytes(idx int, from []byte) error {
	if !vl.isDynamic || vl.alignBytes == 1 {
		err := fmt.Errorf("gpu.Value SetDynamicFromBytes %s: Cannot call this on a non-DynamicOffset Uniform or Storage variable; use SetValueFrom instead", vl.Name)
		return errors.Log(err)
	}
	if idx >= vl.DynamicN {
		err := fmt.Errorf("gpu.Value SetDynamicFromBytes %s: Index: %d >= DynamicN: %d", vl.Name, idx, vl.DynamicN)
		return errors.Log(err)
	}
	sz := vl.MemSize()
	vl.dynamicBuffer = slicesx.SetLength(vl.dynamicBuffer, sz) // preserves data
	nb := len(from)
	if nb != vl.VarSize {
		err := fmt.Errorf("gpu.Value SetDynamicFromBytes %s, Size passed: %d != Size expected %d", vl.Name, nb, vl.VarSize)
		return errors.Log(err)
	}
	off := idx * vl.AlignVarSize
	copy(vl.dynamicBuffer[off:off+vl.VarSize], from)
	return nil
}

// WriteDynamicBuffer writes the staging buffer up to the GPU
// device, after calling SetDynamicValueFrom for all the individual
// dynamic index cases that need to be updated.
// If this is not called, then the data will not be used!
func (vl *Value) WriteDynamicBuffer() error {
	sz := vl.MemSize()
	nb := len(vl.dynamicBuffer)
	if sz != nb {
		err := fmt.Errorf("gpu.Value WriteDynamicBuffer %s, Staging buffer size: %d != Size expected %d; must call SetDynamicValueFrom to establish correct staging buffer size", vl.Name, nb, sz)
		return errors.Log(err)
	}
	if vl.buffer == nil || nb != vl.AllocSize {
		vl.Release()
		buf, err := vl.device.Device.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    vl.Name,
			Contents: vl.dynamicBuffer,
			Usage:    vl.role.BufferUsages(),
		})
		if errors.Log(err) != nil {
			return err
		}
		vl.buffer = buf
		vl.AllocSize = nb
	} else {
		err := vl.device.Queue.WriteBuffer(vl.buffer, 0, vl.dynamicBuffer)
		if errors.Log(err) != nil {
			return err
		}
	}
	return nil
}

// SetDynamicIndex sets the dynamic index to use for
// the current value, returning the value or nil if if the index
// was out of range (logs an error too).
func (vl *Value) SetDynamicIndex(idx int) *Value {
	if idx >= vl.DynamicN {
		slog.Error("gpu.Values.SetDynamicIndex", "index", idx, "is out of range", vl.DynamicN)
		return nil
	}
	vl.DynamicIndex = idx
	return vl
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
	if vr.DynamicOffset {
		return []wgpu.BindGroupEntry{{
			Binding: uint32(vr.Binding),
			Buffer:  vl.buffer,
			Offset:  0,
			Size:    uint64(vl.VarSize), // note: size of one element
		},
		}
	}
	return []wgpu.BindGroupEntry{{
		Binding: uint32(vr.Binding),
		Buffer:  vl.buffer,
		Offset:  0,
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
func (vl *Value) SetFromGoImage(img image.Image, layer int) *TextureSample {
	err := vl.Texture.SetFromGoImage(img, layer)
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

// SetCurrentValue sets the Current value to given index,
// returning the value or nil if if the index
// was out of range (logs an error too).
func (vs *Values) SetCurrentValue(idx int) *Value {
	if idx >= len(vs.Values) {
		slog.Error("gpu.Values.SetCurrentValue", "index", idx, "is out of range", len(vs.Values))
		return nil
	}
	vs.Current = idx
	return vs.CurrentValue()
}

// SetDynamicIndex sets the dynamic index to use for
// the current value, returning the value or nil if if the index
// was out of range (logs an error too).
func (vs *Values) SetDynamicIndex(idx int) *Value {
	vl := vs.CurrentValue()
	return vl.SetDynamicIndex(idx)
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
	vl := vs.CurrentValue()
	return uint32(vl.AlignVarSize * vl.DynamicIndex)
}
