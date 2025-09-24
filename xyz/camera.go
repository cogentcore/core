// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"image"

	"cogentcore.org/core/math32"
)

// Camera defines the properties of the camera
type Camera struct {

	// Pose is the overall orientation and direction of the camera,
	// relative to pointing at negative Z axis with up (positive Y) direction.
	Pose Pose

	// Target location for the camera to point at, which defaults to the origin.
	// This moves with panning movements, and is reset by a call to LookAt method
	Target math32.Vector3

	// UpDir is the upward direction for camera, which defaults to positive Y axis,
	// and is reset by LookAt method.
	UpDir math32.Vector3

	// Ortho switches to using Orthographic instead of default Perspective
	// camera. For Ortho, the view includes the volume specified by the
	// Near - Far distance (i.e., you probably want to decrease Far).
	Ortho bool

	// FOV is field of view in degrees.
	FOV float32

	// Aspect is the aspect ratio (width/height).
	Aspect float32 `set:"-"`

	// Near plane z coordinate.
	Near float32

	// Far plane z coordinate.
	Far float32

	// ViewMatrix is the inverse of the Pose.Matrix.
	ViewMatrix math32.Matrix4 `display:"-"`

	// ProjectionMatrix defines the camera perspective / ortho transform.
	ProjectionMatrix math32.Matrix4 `display:"-"`

	// InvProjectionMatrix is the inverse of the projection matrix.
	InvProjectionMatrix math32.Matrix4 `display:"-"`

	// Frustum of projection: viewable space defined by 6 planes of a pyrammidal shape.
	Frustum *math32.Frustum `display:"-"`
}

func (cm *Camera) Defaults() {
	cm.FOV = 30
	cm.Aspect = 1.5
	cm.Near = .01
	cm.Far = 1000
	cm.DefaultPose()
}

// DefaultPose resets the camera pose to default location and orientation, looking
// at the origin from 0,0,10, with up Y axis
func (cm *Camera) DefaultPose() {
	cm.Pose.Defaults()
	cm.Pose.Pos.Set(0, 0, 10)
	cm.LookAtOrigin()
}

// GenGoSet returns code to set values at given path (var.member etc)
func (cm *Camera) GenGoSet(path string) string {
	return cm.Pose.GenGoSet(path+".Pose") + "; " + cm.Target.GenGoSet(path+".Target") + "; " + cm.UpDir.GenGoSet(path+".UpDir")
}

// SetAspect sets the aspect ratio from the given frame size.
func (cm *Camera) SetAspect(size image.Point) {
	cm.Aspect = float32(size.X) / float32(size.Y)
}

// UpdateMatrix updates the view and projection matricies
func (cm *Camera) UpdateMatrix() {
	cm.Pose.UpdateMatrix()
	cm.ViewMatrix.SetInverse(&cm.Pose.Matrix)
	if cm.Ortho {
		height := 2 * cm.Far * math32.Tan(math32.DegToRad(cm.FOV*0.5))
		width := cm.Aspect * height
		cm.ProjectionMatrix.SetOrthographic(width, height, cm.Near, cm.Far)
	} else {
		cm.ProjectionMatrix.SetPerspective(cm.FOV, cm.Aspect, cm.Near, cm.Far) // use for everything
	}
	cm.InvProjectionMatrix.SetInverse(&cm.ProjectionMatrix)
	var proj math32.Matrix4
	proj.MulMatrices(&cm.ProjectionMatrix, &cm.ViewMatrix)
	cm.Frustum = math32.NewFrustumFromMatrix(&proj)
}

// LookAt points the camera at given target location, using given up direction,
// and sets the Target, UpDir fields for future camera movements.
func (cm *Camera) LookAt(target, upDir math32.Vector3) {
	cm.Target = target
	if upDir == (math32.Vector3{}) {
		upDir = math32.Vec3(0, 1, 0)
	}
	cm.UpDir = upDir
	cm.Pose.LookAt(target, upDir)
	cm.UpdateMatrix()
}

// LookAtOrigin points the camera at origin with Y axis pointing Up (i.e., standard)
func (cm *Camera) LookAtOrigin() {
	cm.LookAt(math32.Vector3{}, math32.Vec3(0, 1, 0))
}

// LookAtTarget points the camera at current target using current up direction
func (cm *Camera) LookAtTarget() {
	cm.LookAt(cm.Target, cm.UpDir)
}

// ViewVector is the vector between the camera position and target
func (cm *Camera) ViewVector() math32.Vector3 {
	return cm.Pose.Pos.Sub(cm.Target)
}

// DistanceTo is the distance from camera to given point
func (cm *Camera) DistanceTo(pt math32.Vector3) float32 {
	dv := cm.Pose.Pos.Sub(pt)
	return dv.Length()
}

// ViewMainAxis returns the dimension along which the view vector is largest
// along with the sign of that axis (+1 for positive, -1 for negative).
// this is useful for determining how manipulations should function, for example.
func (cm *Camera) ViewMainAxis() (dim math32.Dims, sign float32) {
	vv := cm.ViewVector()
	va := vv.Abs()
	switch {
	case va.X > va.Y && va.X > va.Z:
		return math32.X, math32.Sign(vv.X)
	case va.Y > va.X && va.Y > va.Z:
		return math32.Y, math32.Sign(vv.Y)
	default:
		return math32.Z, math32.Sign(vv.Z)
	}
}

// Orbit moves the camera along the given 2D axes in degrees
// (delX = left/right, delY = up/down),
// relative to current position and orientation,
// keeping the same distance from the Target, and rotating the camera and
// the Up direction vector to keep looking at the target.
func (cm *Camera) Orbit(delX, delY float32) {
	ctdir := cm.ViewVector()
	if ctdir == (math32.Vector3{}) {
		ctdir.Set(0, 0, 1)
	}
	dir := ctdir.Normal()

	up := cm.UpDir
	right := cm.UpDir.Cross(dir).Normal()
	// up := dir.Cross(right).Normal() // ensure ortho -- not needed

	// delX rotates around the up vector
	dxq := math32.NewQuatAxisAngle(up, math32.DegToRad(delX))
	dx := ctdir.MulQuat(dxq).Sub(ctdir)
	// delY rotates around the right vector
	dyq := math32.NewQuatAxisAngle(right, math32.DegToRad(delY))
	dy := ctdir.MulQuat(dyq).Sub(ctdir)

	cm.Pose.Pos = cm.Pose.Pos.Add(dx).Add(dy)
	cm.UpDir = cm.UpDir.MulQuat(dyq) // this is only one that affects up

	cm.LookAtTarget()
}

// Pan moves the camera along the given 2D axes (left/right, up/down),
// relative to current position and orientation (i.e., in the plane of the
// current window view)
// and it moves the target by the same increment, changing the target position.
func (cm *Camera) Pan(delX, delY float32) {
	dx := math32.Vec3(-delX, 0, 0).MulQuat(cm.Pose.Quat)
	dy := math32.Vec3(0, -delY, 0).MulQuat(cm.Pose.Quat)
	td := dx.Add(dy)
	cm.Pose.Pos.SetAdd(td)
	cm.Target.SetAdd(td)
}

// PanAxis moves the camera and target along world X,Y axes
func (cm *Camera) PanAxis(delX, delY float32) {
	td := math32.Vec3(-delX, -delY, 0)
	cm.Pose.Pos.SetAdd(td)
	cm.Target.SetAdd(td)
}

// PanTarget moves the target along world X,Y,Z axes and does LookAt
// at the new target location.  It ensures that the target is not
// identical to the camera position.
func (cm *Camera) PanTarget(delX, delY, delZ float32) {
	td := math32.Vec3(-delX, -delY, delZ)
	cm.Target.SetAdd(td)
	dist := cm.ViewVector().Length()
	if dist == 0 {
		cm.Target.SetAdd(td)
	}
	cm.LookAtTarget()
}

// TargetFromView updates the target location from the current view matrix,
// by projecting the current target distance along the current camera
// view matrix.
func (cm *Camera) TargetFromView() {
	trgdist := cm.Pose.Pos.Sub(cm.Target).Length() // distance to existing target
	tpos := math32.Vec4(0, 0, -trgdist, 1)         // target is that distance along -Z axis in front of me
	cm.Target = math32.Vector3FromVector4(tpos.MulMatrix4(&cm.Pose.Matrix))
}

// Zoom moves along axis given pct closer or further from the target
// it always moves the target back also if it distance is < 1
func (cm *Camera) Zoom(zoomPct float32) {
	ctaxis := cm.ViewVector()
	if ctaxis == (math32.Vector3{}) {
		ctaxis.Set(0, 0, 1)
	}
	dist := ctaxis.Length()
	del := ctaxis.MulScalar(zoomPct)
	cm.Pose.Pos.SetAdd(del)
	if zoomPct < 0 && dist < 1 {
		cm.Target.SetAdd(del)
	}
}

// ZoomTo moves along axis in vector pointing through the given 2D point as
// into the camera NDC normalized display coordinates.  Point must be
// 0 normalized, (subtract the Scene ObjBBox.Min) and size of Scene is
// passed as size argument.
// ZoomPct is proportion closer (positive) or further (negative) from the target.
func (cm *Camera) ZoomTo(pt, size image.Point, zoomPct float32) {
	fsize := math32.Vec2(float32(size.X), float32(size.Y))
	fpt := math32.Vec2(float32(pt.X), float32(pt.Y))
	ndc := fpt.WindowToNDC(fsize, math32.Vector2{}, true) // flipY
	ndc.Z = -1                                            // at closest point
	cdir := math32.Vector4FromVector3(ndc, 1).MulMatrix4(&cm.InvProjectionMatrix)
	cdir.Z = -1
	cdir.W = 0 // vec
	// get world position / transform of camera: matrix is inverse of ViewMatrix
	wdir := math32.Vector3FromVector4(cdir.MulMatrix4(&cm.Pose.Matrix))
	del := wdir.MulScalar(zoomPct)
	cm.Pose.Pos.SetAdd(del)
	cm.UpdateMatrix()
	cm.TargetFromView()
}
