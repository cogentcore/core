// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import "cogentcore.org/core/math32"

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
