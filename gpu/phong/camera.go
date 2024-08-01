// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
)

// Camera contains the camera view and projection matricies, for uniform uploading.
type Camera struct {
	// View Camera: transforms world into camera-centered, 3D coordinates.
	View math32.Matrix4

	// Projection Camera: transforms camera coords into 2D render coordinates.
	Projection math32.Matrix4
}

// SetCamera the camera view and projection matrixes, and updates
// uniform data, so they are ready to use.
func (ph *Phong) SetCamera(view, projection *math32.Matrix4) {
	ph.Camera.View = *view
	ph.Camera.Projection = *projection
	vl := ph.Sys.Vars.ValueByIndex(int(CameraGroup), "Camera", 0)
	gpu.SetValueFrom(vl, []Camera{Camera{View: *view, Projection: *projection}})
	ph.cameraUpdated = true
}

// CameraViewMat returns the camera view matrix, based position
// of camera facing at target position, with given up vector.
func CameraViewMat(pos, target, up math32.Vector3) *math32.Matrix4 {
	var lookq math32.Quat
	lookq.SetFromRotationMatrix(math32.NewLookAt(pos, target, up))
	scale := math32.Vec3(1, 1, 1)
	var cview math32.Matrix4
	cview.SetTransform(pos, lookq, scale)
	view, _ := cview.Inverse()
	return view
}
