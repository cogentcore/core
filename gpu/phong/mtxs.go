// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"unsafe"

	"cogentcore.org/core/math32"
)

// Mtxs contains the camera view and projection matricies, for uniform uploading
type Mtxs struct {

	// View Matrix: transforms world into camera-centered, 3D coordinates
	View math32.Matrix4

	// Projection Matrix: transforms camera coords into 2D render coordinates
	Projection math32.Matrix4
}

// SetViewProjection sets the camera view and projection matrixes, and updates
// uniform data, so they are ready to use.
func (ph *Phong) SetViewProjection(view, projection *math32.Matrix4) {
	ph.Cur.VPMtx.View = *view
	ph.Cur.VPMtx.Projection = *projection
	vars := ph.Sys.Vars()
	_, mtx, _ := vars.ValueByIndexTry(int(MtxsSet), "Mtxs", 0)
	mtx.CopyFromBytes(unsafe.Pointer(&ph.Cur.VPMtx))
	ph.Sys.Mem.SyncToGPU()
	vars.BindDynamicValueIndex(int(MtxsSet), "Mtxs", 0)
}

// SetModelMtx sets the model pose matrix -- must be set per render step
// (otherwise last one will be used)
func (ph *Phong) SetModelMtx(model *math32.Matrix4) {
	ph.Cur.ModelMtx = *model
}
