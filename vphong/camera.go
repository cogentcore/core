// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import "github.com/goki/mat32"

// CameraViewMat returns the camera view matrix, based position
// of camera facing at target position, with given up vector.
func CameraViewMat(pos, target, up mat32.Vec3) *mat32.Mat4 {
	var lookq mat32.Quat
	lookq.SetFromRotationMatrix(mat32.NewLookAt(pos, target, up))
	scale := mat32.Vec3{1, 1, 1}
	var cview mat32.Mat4
	cview.SetTransform(pos, lookq, scale)
	view, _ := cview.Inverse()
	return view
}
