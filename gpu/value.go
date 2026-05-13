// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"image"
	"log/slog"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/slicesx"
	"github.com/cogentcore/webgpu/wgpu"
)

// Value represents a specific value of a Var variable, with
// its own WebGPU Buffer or Texture associated with it.
// The Current active Value index can be set in the corresponding Var.Values.
// The Buffer for a Uniform or Storage value is created on the first
// [SetValueFrom] call, or explicitly in [CreateBuffer].
// To read memory back from the GPU, you must do [CreateReadBuffer]
// and use the steps oulined in [GPUToRead].  Use ad-hoc [ValueGroup]s
// to organize these batched read operations efficiently for the different
// groups of values that need to be read back in the different compute stages.
type Value struct {
	// name of this value, named by default as the variable name_idx
	Name string

	// index of this value within the Var list of values
	Index int

	// VarSize is the size of each Var element, which includes any fixed Var.ArrayN
	// array size specified on the Var.
	// The actual buffer size is VarSize * Value.ArrayN (or DynamicN for dynamic).
	VarSize int

	// ArrayN is the actual number of array elements, for Uniform or Storage
	// variables without a fixed array size (i.e., the Var ArrayN = 1).
	// This is set when the buffer is actually created, based on the data,
	// or can be set directly prior to buffer creation.
	ArrayN int

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
	// For dynamic, it is dynamicN * AlignVarSize.
	AllocSize int

	role VarRoles

	// for this variable type, this is the alignment requirement in bytes,
	// for DynamicOffset variables.  This is 1 for Vertex buffer variables.
	alignBytes int

	// true if this is a dynamic variable (Vertex, DynamicOffset Uniform or Storage)
	isDynamic bool

	device Device

	// buffer for this value, makes it accessible to the GPU
	buffer *wgpu.Buffer

	// readBuffer is a CPU-based read buffer for compute values
	// to read data back from the GPU.
	readBuffer *wgpu.Buffer

	// dynamicN is the number of Var elements to encode within one
	// Value buffer, for Vertex values or DynamicOffset values
	// (otherwise it is always effectively 1).
	dynamicN int

	// dynamicBuffer is a CPU-based staging buffer for dynamic values
	// so you can separately set individual dynamic index values and
	// then efficiently copy the entire thing to the device buffer
	// once everything has been set.
	dynamicBuffer []byte

	// for SampledTexture Var roles, this is the Texture.
	// Can set Sampler parameters directly on this.
	Texture *Texture

	// variable for this value
	vvar *Var
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

// MemSizeAlignDown returns the size aligned according to align byte increments,
// rounding down, not up.
func MemSizeAlignDown(size, align int) int {
	if size%align == 0 {
		return size
	}
	nb := size / align
	return nb * align
}

// init initializes value based on variable and index
// within list of vals for this var.
func (vl *Value) init(vr *Var, dev *Device, idx int) {
	vl.vvar = vr
	vl.role = vr.Role
	vl.device = *dev
	vl.Index = idx
	vl.Name = fmt.Sprintf("%s_%d", vr.Name, vl.Index)
	vl.VarSize = vr.MemSize()
	vl.ArrayN = 1
	vl.alignBytes = vr.alignBytes
	vl.AlignVarSize = MemSizeAlign(vl.VarSize, vl.alignBytes)
	vl.isDynamic = vl.role == Vertex || vl.role == Index || vr.DynamicOffset
	vl.dynamicN = 1
	if vr.Role >= SampledTexture {
		vl.Texture = NewTexture(dev)
	}
}

func (vl *Value) String() string {
	return fmt.Sprintf("Bytes: 0x%X", vl.MemSize())
}

// MemSize returns the memory allocation size for this value, in bytes.
func (vl *Value) MemSize() int {
	if vl.Texture != nil {
		return vl.Texture.Format.TotalByteSize()
	}
	if vl.isDynamic {
		return vl.AlignVarSize * vl.dynamicN
	}
	return vl.ArrayN * vl.VarSize
}

// CreateBuffer creates the GPU buffer for this value if it does not
// yet exist or is not the right size.
// For !ReadOnly [Storage] buffers, calls [Value.CreateReadBuffer].
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
		Usage:            vl.vvar.bufferUsages(),
		MappedAtCreation: false,
	})
	if errors.Log(err) != nil {
		return err
	}
	vl.AllocSize = sz
	vl.buffer = buf
	if vl.role == Storage && !vl.vvar.ReadOnly {
		vl.CreateReadBuffer()
	}
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
	vl.ReleaseRead()
}

// ReleaseRead releases the read buffer
func (vl *Value) ReleaseRead() {
	if vl.readBuffer != nil {
		vl.readBuffer.Release()
		vl.readBuffer = nil
	}
}

// NilBufferCheckCheck checks if buffer is nil, returning error if so
func (vl *Value) NilBufferCheck() error {
	if vl.buffer == nil {
		return fmt.Errorf("gpu.Value NilBufferCheck: buffer is nil for value: %s", vl.Name)
	}
	return nil
}

// DynamicN returns the number of dynamic values currently configured.
func (vl *Value) DynamicN() int {
	return vl.dynamicN
}

// varGroupDirty tells our VarGroup that our buffers have changed,
// so a new bindGroup needs to be created when it is next requested.
func (vl *Value) varGroupDirty() {
	vl.vvar.VarGroup.ValuesUpdated()
}

// SetDynamicN sets the number of dynamic values for this Value.
// If different, a new bindgroup must be generated.
func (vl *Value) SetDynamicN(n int) {
	if n == vl.dynamicN {
		return
	}
	vl.varGroupDirty()
	vl.dynamicN = n
}

// SetValueFrom copies given values into value buffer memory,
// making the buffer if it has not yet been constructed.
// The actual ArrayN size of Storage or Uniform variables will
// be computed based on the size of the from bytes, relative to
// the variable size.
// IMPORTANT: do not use this for dynamic offset Uniform or
// Storage variables, as the alignment will not be correct;
// See [SetDynamicFromBytes].
func SetValueFrom[E any](vl *Value, from []E) error {
	return vl.SetFromBytes(wgpu.ToBytes(from))
}

// SetFromBytes copies given bytes into value buffer memory,
// making the buffer if it has not yet been constructed.
// For !ReadOnly [Storage] buffers, calls [Value.CreateReadBuffer].
// IMPORTANT: do not use this for dynamic offset Uniform or
// Storage variables, as the alignment will not be correct;
// See [SetDynamicFromBytes].
func (vl *Value) SetFromBytes(from []byte) error {
	if vl.isDynamic && vl.alignBytes != 1 {
		err := fmt.Errorf("gpu.Value SetFromBytes %s: Cannot call this on a DynamicOffset Uniform or Storage variable; use SetDynamicValueFrom instead", vl.Name)
		return errors.Log(err)
	}
	nb := len(from)
	an := nb / vl.VarSize
	aover := nb % vl.VarSize
	if aover != 0 {
		err := fmt.Errorf("gpu.Value SetFromBytes %s, Size passed: %d is not an even multiple of the variable size: %d", vl.Name, nb, vl.VarSize)
		return errors.Log(err)
	}
	if vl.isDynamic { // Vertex, Index at this point
		vl.SetDynamicN(an)
	} else {
		vl.ArrayN = an
	}
	tb := vl.MemSize()
	if nb != tb { // this should never happen, but justin case
		err := fmt.Errorf("gpu.Value SetFromBytes %s, Size passed: %d != Size expected %d", vl.Name, nb, tb)
		return errors.Log(err)
	}
	if vl.buffer == nil || vl.AllocSize != tb {
		vl.varGroupDirty() // only if tb is different; buffer created in bindGroupEntry so not nil here
		vl.Release()
		buf, err := vl.device.Device.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    vl.Name,
			Contents: from,
			Usage:    vl.vvar.bufferUsages(),
		})
		if errors.Log(err) != nil {
			return err
		}
		vl.buffer = buf
		vl.AllocSize = nb
		if vl.role == Storage && !vl.vvar.ReadOnly {
			vl.CreateReadBuffer()
		}
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
// Must call [WriteDynamicBuffer] after all such values have been updated,
// to actually copy the entire staging buffer data to the GPU device.
// Vertex variables must have separate values for each, and do not
// support dynamic indexing.
// It is essential that [DynamicN] is set properly before
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
	if idx >= vl.dynamicN {
		err := fmt.Errorf("gpu.Value SetDynamicFromBytes %s: Index: %d >= DynamicN: %d", vl.Name, idx, vl.dynamicN)
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
		vl.varGroupDirty() // only if nb is different; buffer created in bindGroupEntry so not nil here
		vl.Release()
		buf, err := vl.device.Device.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    vl.Name,
			Contents: vl.dynamicBuffer,
			Usage:    vl.vvar.bufferUsages(),
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
	if idx >= vl.dynamicN {
		slog.Error("gpu.Values.SetDynamicIndex", "index", idx, "is out of range", vl.dynamicN)
		return nil
	}
	vl.DynamicIndex = idx
	return vl
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
// The Sampler is also configured at this point, with the current settings,
// so set those before making this call.  It will not be re-configured
// without manually releasing it.
func (vl *Value) SetFromGoImage(img image.Image, layer int) *Texture {
	err := vl.Texture.SetFromGoImage(img, layer)
	errors.Log(err)
	err = vl.Texture.Sampler.Config(&vl.device)
	errors.Log(err)
	return vl.Texture
}

// SetFromTexture sets Texture from an existing gpu Texture.
// The Sampler is also ensured configured at this point,
// with the current settings, so set those before making this call.
// It will not be re-configured without manually releasing it.
func (vl *Value) SetFromTexture(tx *Texture) *Texture {
	vl.Texture.SetShared(tx)
	err := vl.Texture.Sampler.Config(&vl.device)
	errors.Log(err)
	return vl.Texture
}

// CreateReadBuffer creates a read buffer for this value,
// for [Storage] values only. Automatically called for !ReadOnly.
// Read buffer is needed for reading values back from the GPU.
func (vl *Value) CreateReadBuffer() error {
	if !(vl.role == Storage || vl.role == StorageTexture) || vl.vvar.ReadOnly {
		return nil
	}
	sz := vl.MemSize()
	if sz == 0 {
		vl.ReleaseRead()
		return nil
	}
	if sz == vl.AllocSize && vl.readBuffer != nil {
		return nil
	}
	vl.ReleaseRead()

	buf, err := vl.device.Device.CreateBuffer(&wgpu.BufferDescriptor{
		Size:             uint64(sz),
		Label:            vl.Name,
		Usage:            wgpu.BufferUsageMapRead | wgpu.BufferUsageCopyDst,
		MappedAtCreation: false,
	})
	if errors.Log(err) != nil {
		return err
	}
	vl.readBuffer = buf
	return nil
}

func (vl *Value) readNilCheck() error {
	if vl.readBuffer == nil {
		return fmt.Errorf("gpu.Value %q: read buffer is nil", vl.Name)
	}
	return nil
}

// GPUToRead adds commands to the given command encoder
// to copy given value from its GPU buffer to the read buffer,
// which must have already been created.
// This is the first step for reading values back from the GPU,
// which starts with this command to be executed in the compute pass,
// and then requires ReadSync to actually read the data into
// the CPU side of the read buffer from the GPU, and ends with
// a final ReadToBytes call to copy the raw read bytes into
// a target data structure.
func (vl *Value) GPUToRead(cmd *wgpu.CommandEncoder) error {
	if err := vl.readNilCheck(); err != nil {
		return errors.Log(err)
	}
	return cmd.CopyBufferToBuffer(vl.buffer, 0, vl.readBuffer, 0, uint64(vl.AllocSize))
}

// ReadSync reads data from GPU to CPU side of the read buffer.
// It is much more efficient to call [ValueReadSync] with _all_ values that
// need to be sync'd at the same time, so only use this when copying one value.
// See [GPUToRead] for overview of the process.
func (vl *Value) ReadSync() error {
	if err := vl.readNilCheck(); err != nil {
		return errors.Log(err)
	}
	return ValueReadSync(&vl.device, vl)
}

// ReadToBytes copies value read buffer data into
// the memory bytes occupied by the given object.
// You must have called [ReadSync] on the value
// prior to calling this, so that the memory is mapped.
// This automatically calls Unmap() after copying,
// which is essential for being able to re-use the read buffer again.
func ReadToBytes[E any](vl *Value, dest []E) error {
	return vl.ReadToBytes(wgpu.ToBytes(dest))
}

// ReadToBytes copies data from read buffer into given byte slice
// which is enforced to be the correct size if not already.
func (vl *Value) ReadToBytes(to []byte) error {
	if err := vl.readNilCheck(); err != nil {
		return errors.Log(err)
	}
	copy(to, vl.readBuffer.GetMappedRange(0, uint(vl.AllocSize)))
	vl.readBuffer.Unmap()
	return nil
}
