// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"log"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/slicesx"
	"github.com/cogentcore/webgpu/wgpu"
)

// Values is a list container of Value values, accessed by index or name.
type Values struct {
	// values in indexed order.
	Values []*Value

	// current specifies the current value to use in rendering.
	current int

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
	return vs.Values[vs.current]
}

// SetCurrentValue sets the Current value to given index,
// returning the value or nil if if the index
// was out of range (logs an error too).
func (vs *Values) SetCurrentValue(idx int) (*Value, error) {
	if idx >= len(vs.Values) {
		err := fmt.Errorf("gpu.Values.SetCurrentValue index %d is out of range %d", idx, len(vs.Values))
		errors.Log(err)
		return nil, err
	}
	if vs.current != idx {
		vs.current = idx
		vs.Values[0].vvar.VarGroup.ValuesUpdated()
	}
	return vs.CurrentValue(), nil
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
	vl, err := vs.ValueByIndex(idx)
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
func (vs *Values) ValueByIndex(idx int) (*Value, error) {
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
func (vs *Values) ValueByName(name string) (*Value, error) {
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
