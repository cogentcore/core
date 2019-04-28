// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import "github.com/goki/gi/mat32"

// Camera defines the properties of the camera
type Camera struct {
	Pose       Pose       `desc:"overall orientation and direction of the camera, relative to pointing at negative Z axis with up direction"`
	Ortho      bool       `desc:"default is a Perspective camera -- set this to make it Orthographic instead, in which case the view includes the volume specified by the Far distance."`
	FOV        float32    `desc:"field of view in degrees "`
	Aspect     float32    `desc:"aspect ratio (width/height)"`
	Near       float32    `desc:"near plane z coordinate"`
	Far        float32    `desc:"far plane z coordinate"`
	ViewMatrix mat32.Mat4 `desc:"view matrix (inverse of the Pose.Matrix)"`
	PrjnMatrix mat32.Mat4 `desc:"projection matrix, defining the camera perspective / ortho transform"`
}

func (cm *Camera) Defaults() {
	cm.Pose.Pos.Set(0, 0, -10)
	cm.LookAt(&mat32.Vec3{0, 0, 0}, &mat32.Vec3{0, 1, 0})
}

// UpdateMatrix updates the view and prjn matricies
func (cm *Camera) UpdateMatrix() {
	cm.Pose.UpdateMatrix()
	cm.ViewMatrix.GetInverse(&cm.Pose.Matrix)
	if cm.Ortho {
		height := 2 * cm.Far * mat32.Tan(mat32.DegToRad(cm.FOV*0.5))
		width := cm.Aspect * height
		cm.PrjnMatrix.MakeOrthographic(width, height, cm.Near, cm.Far)
	} else {
		cm.PrjnMatrix.MakePerspective(cm.FOV, cm.Aspect, cm.Near, cm.Far)
	}
}

// LookAt points the camera at given target location, using given up direction
// updates the internal Quat rotation vector
func (cm *Camera) LookAt(target, upDir *mat32.Vec3) {
	rotMat := mat32.Mat4{}
	rotMat.LookAt(&cm.Pose.Pos, target, upDir)
	cm.Pose.Quat.SetFromRotationMatrix(&rotMat)
}
