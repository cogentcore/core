/ Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"image"
	"log"
	"log/slog"
	"unsafe"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/gpu/szalloc"
	"cogentcore.org/core/math32"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// Value represents a specific value of a Var variable, with
// its own WebGPU Buffer or Texture associated with it.
// The current active Value can be set on the corresponding Var.
// Typically there are only multiple values for Vertex and Texture vars.
type Value struct {
	// name of this value, named by default as the variable name_idx
	Name string

	// index of this value within the Var list of values
	Index int

	// actual number of elements in an array, where 1 means scalar / singular value.
	// If 0, this is a dynamically sized item and the size must be set.
	N int

	// if N > 1 (array) then this is the effective size of each element,
	// which must be aligned to 16 byte modulo for Uniform types.
	// Non-naturally aligned types require slower element-by-element
	// syncing operations, instead of memcopy.
	ElSize int

	// total memory size of this value in bytes, as allocated in buffer.
	AllocSize int

	device Device
	
	// buffer for this value, makes it accessible to the GPU
	buffer *wgpu.Buffer `display:"-"`

	// for SampledTexture Var roles, this is the Texture.
	texture *Texture
	
	TextureOwns bool
}

func NewValue(vr *Var, dev *Device, idx int) *Value {
	vl := &Value{}
	vl.Init(vr, dev, idx)
	return vl
}

// Init initializes value based on variable and index
// within list of vals for this var.
func (vl *Value) Init(vr *Var, dev *Device, idx int) {
	vl.Device = *dev
	vl.Index = idx
	vl.Name = fmt.Sprintf("%s_%d", vr.Name, vl.Index)
	vl.ElSize = vr.SizeOf
	vl.N = vr.ArrayN
	vl.TextureOwns = vr.TextureOwns
	if vr.Role >= SampledTexture {
		vl.texture = NewTexture(dev)
	} else {
		vl.CreateBuffer(vr, dev)
	}
}

// Size returns the memory allocation size for this value, in bytes.
func (vl *Value) Size() int {
	if vl.N == 0 {
		vl.N = 1
	}
	if vr.Role == SampledTexture {
		if vl.TextureOwns {
			return 0
		} else {
			return vl.texture.Format.TotalByteSize()
		}
	} else {
		return vl.ElSize * vl.N
	}
}

// CreateBuffer creates the GPU buffer for this value if it does not
// yet exist or is not the right size.
// Buffers always start mapped.
func (vl *Value) CreateBuffer(vr *Var, dev *Device) error {
	if vr.Role == SampledTexture {
		return nil
	}
	sz := vl.Size()
	if sz == 0 {
		vl.Release()
		return nil
	}
	if sz == vl.AllocSize && vl.buffer != nil {
		return nil
	}
	vl.Release()
	buf, err := dev.Device.CreateBuffer(&wgpu.BufferDescriptor{
		Size:             uint64(sz),
		Label:            Name,
		Usage:            vr.Role.BufferUsages(),
		MappedAtCreation: true,
	})
	if err != nil {
		slog.Error(err)
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
	if vl.texture != nil {
		vl.texture.Release()
		vl.texture = nil
	}
}

// NilBufferCheckCheck checks if buffer is nil, returning error if so
func (vl *Value) NilBufferCheck() error {
	if vl.buffer == nil {
		return fmt.Errorf("gpu.Value NilBufferCheck: buffer is nil for value: %s", vl.Name)
	}
	return nil
}

// SetValueFromAsync copies given values into value buffer memory,
// ensuring that the buffer is mapped and ready to be copied into.
// This automatically calls Unmap() after copying.
func SetValueFromAsync[E any](vl *Value, from []E) error {
	return vl.SetFromBytesAsync(wgpu.ToBytes(from))
}

// SetFromBytesAsync copies given bytes into value buffer memory,
// ensuring that the buffer is mapped and ready to be copied into.
// This automatically calls Unmap() after copying.
func (vl *Value) SetFromBytesAsync(from []byte) error {
	if err := vl.NilBufferCheck(); err != nil {
		slog.Error(err)
		return err
	}
	vl.buffer.MapAsync(wgpu.MapMode_Write, 0, vl.AllocSize, func(stat BufferMapAsyncStatus) {
		if stat != wgpu.BufferMapAsyncStatus_Success {
			err = return fmt.Errorf("gpu.Value SetFromBytesAsync: %s for value: %s", stat.String(), vl.Name)
			return
		}
		bm := vl.buffer.GetMappedRange(0, vl.AllocSize)
		copy(bm, from)
		vl.buffer.Unmap()
	})
	return err
}

// CopyValueToBytesAsync copies given value buffer memory to given bytes,
// ensuring that the buffer is mapped and ready to be copied into.
// This automatically calls Unmap() after copying.
func CopyValueToBytesAsync[E any](vl *Value, dest []E) error {
	return vl.CopyToBytesAsync(wgpu.ToBytes(dest))
}

// CopyToBytesAsync copies value buffer memory to given bytes,
// ensuring that the buffer is mapped and ready to be copied into.
// This automatically calls Unmap() after copying.
func (vl *Value) CopyToBytesAsync(dest []byte) error {
	if err := vl.NilBufferCheck(); err != nil {
		slog.Error(err)
		return err
	}
	vl.buffer.MapAsync(wgpu.MapMode_Read, 0, vl.AllocSize, func(stat BufferMapAsyncStatus) {
		if stat != wgpu.BufferMapAsyncStatus_Success {
			err = return fmt.Errorf("gpu.Value CopyToBytesAsync: %s for value: %s", stat.String(), vl.Name)
			return
		}
		bm := vl.buffer.GetMappedRange(0, vl.AllocSize)
		copy(dest, bm)
		vl.buffer.Unmap()
	})
	return err
}

func (vl *Value) BindGroupEntry(vr *Var) []wgpu.BindGroupEntry {
	if vr.Role >= SampledTexture {
		return []wgpu.BindGroupEntry{
			{
				Binding:     vr.Binding,
				TextureView: vl.texture.View
			},
			{
				Binding: vr.Binding+1,
				Sampler: vl.texture.Sampler,
			},
		}
	}
	return []wgpu.BindGroupEntry{{
			Binding: vr.Binding,
			Buffer:  vl.buffer,
			Size:    wgpu.WholeSize,
		},
	}
}

// SetGoImage sets Texture image data from an image.Image standard Go image,
// at given layer. This is most efficiently done using an image.RGBA, but other
// formats will be converted as necessary.
// If flipY is true then the Texture Y axis is flipped when copying into
// the image data.  Can avoid this by configuring texture coordinates to
// compensate.
func (vl *Value) SetGoImage(img image.Image, layer int, flipY bool) error {
	return vl.texture.SetGoImage(img, layer, flipY)
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
		vl.Name = name
		vs.NameMap[name] = vl
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

// Free frees all the value buffers / textures
func (vs *Values) Free() {
	for _, vl := range vs.Values {
		vl.Free()
	}
}

// Release frees all existing values and resets the list of Values so subsequent
// Config will start fresh (e.g., if Var type changes).
func (vs *Values) Release() {
	vs.Free()
	vs.Values = nil
	vs.NameMap = nil
}

// MemSize returns size in bytes across all Values in list
func (vs *Values) MemSize() int {
	tsz := 0
	for _, vl := range vs.Values {
		tsz += vl.Size()
	}
	return tsz
}

// BindGroupEntry returns the BindGroupEntry for Current
// value for this variable.
func (vs *Values) BindGroupEntry(vr *Var) wgpu.BindGroupEntry {
	vl := vs.Values[vs.Current]
	return vl.BindGroupEntry(vr)
}

