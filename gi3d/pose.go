// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import "github.com/goki/gi/mat32"

// Pose contains the full specification of a given object's position and orientation
type Pose struct {
	Pos         mat32.Vec3 `desc:"position of center of object"`
	Scale       mat32.Vec3 `desc:"scale (relative to parent)"`
	Quat        mat32.Quat `desc:"Node rotation specified as a Quat (relative to parent)"`
	Matrix      mat32.Mat4 `desc:"Local matrix. Contains all position/rotation/scale information (relative to parent)"`
	WorldMatrix mat32.Mat4 `desc:"World matrix. Contains all absolute position/rotation/scale information (i.e. relative to very top parent, generally the scene)"`
	MVMatrix    mth32.Mat4 `desc:"model * view matrix -- tranforms into camera-centered coords"`
	MVPMatrix   mth32.Mat4 `desc:"model * view * projection matrix -- full final render matrix"`
}

// UpdateMatrix updates the local transform matrix based on its position, quaternion, and scale.
func (ps *Pose) UpdateMatrix() {
	ps.Matrix.Compose(&ps.Pos, &ps.Quat, &ps.Scale)
}

// UpdateWorldMatrix updates the world transform matrix based on Matrix and parent's WorldMatrix.
// Also calls UpdateMatrix
func (ps *Pose) UpdateWorldMatrix(parWorld *mat32.Mat4) {
	ps.UpdateMatrix()
	ps.WorldMatrix.MultiplyMatricies(parWorld, &ps.Matrix)
}

// UpdateMVPMatrix updates the model * view, * projection matricies based on camera view, prjn matricies
// Assumes that WorldMatrix has been updated
func (ps *Pose) UpdateMVPMatrix(viewMat, prjnMat *mat32.Mat4) {
	ps.MVMatrix.MultiplyMatricies(viewMat, &ps.WorldMatrix)
	ps.MVPMatrix.MultiplyMatricies(prjnMat, &ps.MVMatrix)
}

// MoveOnAxis moves (translates) the specified distance on the specified local axis.
func (ps *Pose) MoveOnAxis(x, y, z, dist float32) {
	v := mat32.NewVec3(x, y, z)
	v.ApplyQuat(&ps.Quat)
	v.MultiplyScalar(dist)
	ps.Pos.Add(v)
}

// SetEulerRotation sets the rotation in Euler angles (radians).
func (ps *Pose) SetEulerRotation(x, y, z float32) {
	rot := mat32.NewVec3(x, y, z)
	ps.Quat.SetFromEuler(rot)
}

// EulerRotation returns the current rotation in Euler angles (radians).
func (ps *Pose) EulerRotation() mat32.Vec3 {
	rot := mat32.Vec3{}
	rot.SetFromQuat(&ps.Quat)
	return rot
}

// SetAxisRotation sets rotation from local axis and angle in radians.
func (ps *Pose) SetAxisRotation(x, y, z, angle float32) {
	axis := mat32.NewVec3(x, y, z)
	ps.Quat.SetFromAxisAngle(&axis, angle)
}

// RotateOnAxis rotates around the specified local axis the specified angle in radians.
func (ps *Pose) RotateOnAxis(x, y, z, angle float32) {
	axis := mat32.NewVec3(x, y, z)
	rotQuat := &mat32.Quat{}
	rotQuat.SetFromAxisAngle(axis, angle)
	ps.Quat.Multiply(&rotQuat)
}

// SetMatrix sets the local transformation matrix and updates Pos, Scale, Quat.
func (ps *Pose) SetMatrix(m *mat32.Mat4) {
	ps.Matrix = *m
	ps.Matrix.Decompose(&ps.Pos, &ps.Quat, &ps.Scale)
}

// WorldPos returns the current world position.
func (ps *Pose) WorldPos() mat32.Vec3 {
	pos := mat32.Vec3{}
	pos.SetFromMatrixPosition(&ps.WorldMatrix)
	return pos
}

// WorldQuat returns the current world quaternion.
func (ps *Pose) WorldQuat() mat32.Quat {
	pos := mat32.Vec3{}
	scale := mat32.Vec3{}
	quat := mat32.Quat{}
	ps.WorldMatrix.Decompose(&pos, &quat, &scale)
	return quat
}

// WorldRotation returns the current world rotation in Euler angles.
func (ps *Pose) WorldRotation() mat32.Vec3 {
	quat := mat32.Quat{}
	ps.WorldQuat(&quat)
	ang := mat32.Vec3{}
	ang.SetFromQuat(&quat)
	return ang
}

// WorldScale returns he current world scale.
func (ps *Pose) WorldScale() mat32.Vec3 {
	pos := mat32.Vec3{}
	scale := mat32.Vec3{}
	quat := mat32.Quat{}
	ps.WorldMatrix.Decompose(&pos, &quat, &scale)
	return scale
}
