// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"fmt"
	"log"
	"unsafe"

	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
)

// Mtxs are projection matrixes per object
type Mtxs struct {
	MVMtx   mat32.Mat4 `desc:"Model * Camera View Matrix"`
	MVPMtx  mat32.Mat4 `desc:"Model * Camera View * Projection Matrix"`
	NormMtx mat32.Mat4 `desc:"Model * View Matrix for Normals"`
}

// SetMtxs sets the matricies based on input matrixes
func (mx *Mtxs) SetMtxs(model, view, prjn *mat32.Mat4) {
	mx.MVMtx.MulMatrices(view, model)
	m3 := mat32.Mat3{}
	m3.SetNormalMatrix(&mx.MVMtx)
	mx.NormMtx.SetFromMat3(&m3)
	mx.MVPMtx.MulMatrices(prjn, &mx.MVMtx)
}

func NewMtxs(model, view, prjn *mat32.Mat4) *Mtxs {
	mtx := &Mtxs{}
	mtx.SetMtxs(model, view, prjn)
	return mtx
}

// AddMtxs adds to list of mtxs
func (ph *Phong) AddMtxs(name string, model, view, prjn *mat32.Mat4) {
	ph.Mtxs.Add(name, NewMtxs(model, view, prjn))
}

// AllocMtxs allocates vals for mtxs
func (ph *Phong) AllocMtxs() {
	vars := ph.Sys.Vars()
	mtxset := vars.SetMap[int(MtxsSet)]
	mtxset.ConfigVals(ph.Mtxs.Len())
}

// ConfigMtxs configures the rendering for the mtxs that have been added.
func (ph *Phong) ConfigMtxs() {
	vars := ph.Sys.Vars()
	mtxset := vars.SetMap[int(MtxsSet)]
	for i, kv := range ph.Mtxs.Order {
		_, mtx, _ := mtxset.ValByIdxTry("Mtxs", i)
		mtx.CopyBytes(unsafe.Pointer(kv.Val))
	}
}

// SetMtxsIdx sets updated mtxs by index (sets Mod, will update with Sync)
func (ph *Phong) SetMtxsIdx(idx int, model, view, prjn *mat32.Mat4) error {
	vars := ph.Sys.Vars()
	_, vl, _ := vars.ValByIdxTry(int(MtxsSet), "Mtxs", idx)
	mtx := NewMtxs(model, view, prjn)
	ph.Mtxs.Order[idx].Val = mtx
	vl.CopyBytes(unsafe.Pointer(mtx))
	return nil
}

// SetMtxsName sets updated mtxs by name (sets Mod, will update with Sync)
func (ph *Phong) SetMtxsName(name string, model, view, prjn *mat32.Mat4) error {
	idx, ok := ph.Mtxs.IdxByKey(name)
	if !ok {
		err := fmt.Errorf("vphong:UseMtxsName -- name not found: %s", name)
		if vgpu.TheGPU.Debug {
			log.Println(err)
		}
	}
	return ph.SetMtxsIdx(idx, model, view, prjn)
}

// UseMtxsIdx selects mtxs by index for current render step
func (ph *Phong) UseMtxsIdx(idx int) error {
	vars := ph.Sys.Vars()
	vars.BindDynValIdx(int(MtxsSet), "Mtxs", idx)
	ph.Cur.MtxsIdx = idx // todo: range check
	return nil
}

// UseMtxsName selects mtxs by name for current render step
func (ph *Phong) UseMtxsName(name string) error {
	idx, ok := ph.Mtxs.IdxByKey(name)
	if !ok {
		err := fmt.Errorf("vphong:UseMtxsName -- name not found: %s", name)
		if vgpu.TheGPU.Debug {
			log.Println(err)
		}
	}
	return ph.UseMtxsIdx(idx)
}
