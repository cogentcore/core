// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
)

// Matrix contains the camera view and projection matricies, for uniform uploading
type Matrix struct {

	// View Matrix: transforms world into camera-centered, 3D coordinates
	View math32.Matrix4

	// Projection Matrix: transforms camera coords into 2D render coordinates
	Projection math32.Matrix4
}

// SetViewProjection sets the camera view and projection matrixes, and updates
// uniform data, so they are ready to use.
func (ph *Phong) SetViewProjection(view, projection *math32.Matrix4) {
	vl := ph.Sys.Vars.ValueByIndex(int(MatrixGroup), "Matrix", 0)
	gpu.SetValueFrom(vl, []Matrix{Matrix{View: *view, Projection: *projection}})
}

// SetModelMtx sets the model pose matrix -- must be set per render step
// (otherwise last one will be used)
func (ph *Phong) SetModelMtx(model *math32.Matrix4) {
	// todo:
	// ph.Cur.ModelMtx = *model
}
