// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"unsafe"

	"goki.dev/mat32/v2"
)

// Mtxs contains the camera view and projection matricies, for uniform uploading
type Mtxs struct {

	// View Matrix: transforms world into camera-centered, 3D coordinates
	View mat32.Mat4 `desc:"View Matrix: transforms world into camera-centered, 3D coordinates"`

	// Projection Matrix: transforms camera coords into 2D render coordinates
	Prjn mat32.Mat4 `desc:"Projection Matrix: transforms camera coords into 2D render coordinates"`
}

// SetViewPrjn sets the camera view and projection matrixes, and updates
// uniform data, so they are ready to use.
func (ph *Phong) SetViewPrjn(view, prjn *mat32.Mat4) {
	ph.Cur.VPMtx.View = *view
	ph.Cur.VPMtx.Prjn = *prjn
	vars := ph.Sys.Vars()
	_, mtx, _ := vars.ValByIdxTry(int(MtxsSet), "Mtxs", 0)
	mtx.CopyFromBytes(unsafe.Pointer(&ph.Cur.VPMtx))
	ph.Sys.Mem.SyncToGPU()
	vars.BindDynValIdx(int(MtxsSet), "Mtxs", 0)
}

// SetModelMtx sets the model pose matrix -- must be set per render step
// (otherwise last one will be used)
func (ph *Phong) SetModelMtx(model *mat32.Mat4) {
	ph.Cur.ModelMtx = *model
}
