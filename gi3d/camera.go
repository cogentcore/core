// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"image"
	"sync"

	"github.com/goki/gi/icons"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Camera defines the properties of the camera
type Camera struct {

	// overall orientation and direction of the camera, relative to pointing at negative Z axis with up (positive Y) direction
	Pose Pose `desc:"overall orientation and direction of the camera, relative to pointing at negative Z axis with up (positive Y) direction"`

	// mutex protecting camera data
	CamMu sync.RWMutex `desc:"mutex protecting camera data"`

	// target location for the camera -- where it is pointing at -- defaults to the origin, but moves with panning movements, and is reset by a call to LookAt method
	Target mat32.Vec3 `desc:"target location for the camera -- where it is pointing at -- defaults to the origin, but moves with panning movements, and is reset by a call to LookAt method"`

	// up direction for camera -- which way is up -- defaults to positive Y axis, and is reset by call to LookAt method
	UpDir mat32.Vec3 `desc:"up direction for camera -- which way is up -- defaults to positive Y axis, and is reset by call to LookAt method"`

	// default is a Perspective camera -- set this to make it Orthographic instead, in which case the view includes the volume specified by the Near - Far distance (i.e., you probably want to decrease Far).
	Ortho bool `desc:"default is a Perspective camera -- set this to make it Orthographic instead, in which case the view includes the volume specified by the Near - Far distance (i.e., you probably want to decrease Far)."`

	// field of view in degrees
	FOV float32 `desc:"field of view in degrees "`

	// aspect ratio (width/height)
	Aspect float32 `desc:"aspect ratio (width/height)"`

	// near plane z coordinate
	Near float32 `desc:"near plane z coordinate"`

	// far plane z coordinate
	Far float32 `desc:"far plane z coordinate"`

	// view matrix (inverse of the Pose.Matrix)
	ViewMatrix mat32.Mat4 `view:"-" desc:"view matrix (inverse of the Pose.Matrix)"`

	// projection matrix, defining the camera perspective / ortho transform
	PrjnMatrix mat32.Mat4 `view:"-" desc:"projection matrix, defining the camera perspective / ortho transform"`

	// vulkan projection matrix -- required for vgpu -- produces same effect as PrjnMatrix, which should be used for all other math
	VkPrjnMatrix mat32.Mat4 `view:"-" desc:"vulkan projection matrix -- required for vgpu -- produces same effect as PrjnMatrix, which should be used for all other math"`

	// inverse of the projection matrix
	InvPrjnMatrix mat32.Mat4 `view:"-" desc:"inverse of the projection matrix"`

	// frustum of projection -- viewable space defined by 6 planes of a pyrammidal shape
	Frustum *mat32.Frustum `view:"-" desc:"frustum of projection -- viewable space defined by 6 planes of a pyrammidal shape"`
}

var TypeCamera = kit.Types.AddType(&Camera{}, CameraProps)

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

// UpdateMatrix updates the view and prjn matricies
func (cm *Camera) UpdateMatrix() {
	cm.CamMu.Lock()
	defer cm.CamMu.Unlock()

	cm.Pose.UpdateMatrix()
	cm.ViewMatrix.SetInverse(&cm.Pose.Matrix)
	if cm.Ortho {
		height := 2 * cm.Far * mat32.Tan(mat32.DegToRad(cm.FOV*0.5))
		width := cm.Aspect * height
		cm.PrjnMatrix.SetOrthographic(width, height, cm.Near, cm.Far)
	} else {
		cm.PrjnMatrix.SetPerspective(cm.FOV, cm.Aspect, cm.Near, cm.Far)     // use for everything
		cm.VkPrjnMatrix.SetVkPerspective(cm.FOV, cm.Aspect, cm.Near, cm.Far) // Vk use for render
	}
	cm.InvPrjnMatrix.SetInverse(&cm.PrjnMatrix)
	var proj mat32.Mat4
	proj.MulMatrices(&cm.PrjnMatrix, &cm.ViewMatrix)
	cm.Frustum = mat32.NewFrustumFromMatrix(&proj)
}

// LookAt points the camera at given target location, using given up direction,
// and sets the Target, UpDir fields for future camera movements.
func (cm *Camera) LookAt(target, upDir mat32.Vec3) {
	cm.CamMu.Lock()
	cm.Target = target
	if upDir.IsNil() {
		upDir = mat32.Vec3Y
	}
	cm.UpDir = upDir
	cm.Pose.LookAt(target, upDir)
	cm.CamMu.Unlock()
	cm.UpdateMatrix()
}

// LookAtOrigin points the camera at origin with Y axis pointing Up (i.e., standard)
func (cm *Camera) LookAtOrigin() {
	cm.LookAt(mat32.Vec3Zero, mat32.Vec3Y)
}

// LookAtTarget points the camera at current target using current up direction
func (cm *Camera) LookAtTarget() {
	cm.LookAt(cm.Target, cm.UpDir)
}

// ViewVector is the vector between the camera position and target
func (cm *Camera) ViewVector() mat32.Vec3 {
	cm.CamMu.RLock()
	defer cm.CamMu.RUnlock()
	return cm.Pose.Pos.Sub(cm.Target)
}

// DistTo is the distance from camera to given point
func (cm *Camera) DistTo(pt mat32.Vec3) float32 {
	cm.CamMu.RLock()
	defer cm.CamMu.RUnlock()
	dv := cm.Pose.Pos.Sub(pt)
	return dv.Length()
}

// ViewMainAxis returns the dimension along which the view vector is largest
// along with the sign of that axis (+1 for positive, -1 for negative).
// this is useful for determining how manipulations should function, for example.
func (cm *Camera) ViewMainAxis() (dim mat32.Dims, sign float32) {
	vv := cm.ViewVector()
	va := vv.Abs()
	switch {
	case va.X > va.Y && va.X > va.Z:
		return mat32.X, mat32.Sign(vv.X)
	case va.Y > va.X && va.Y > va.Z:
		return mat32.Y, mat32.Sign(vv.Y)
	default:
		return mat32.Z, mat32.Sign(vv.Z)
	}
}

// Orbit moves the camera along the given 2D axes in degrees
// (delX = left/right, delY = up/down),
// relative to current position and orientation,
// keeping the same distance from the Target, and rotating the camera and
// the Up direction vector to keep looking at the target.
func (cm *Camera) Orbit(delX, delY float32) {
	ctdir := cm.ViewVector()
	if ctdir.IsNil() {
		ctdir.Set(0, 0, 1)
	}
	dir := ctdir.Normal()

	cm.CamMu.Lock()
	up := cm.UpDir
	right := cm.UpDir.Cross(dir).Normal()
	// up := dir.Cross(right).Normal() // ensure ortho -- not needed

	// delX rotates around the up vector
	dxq := mat32.NewQuatAxisAngle(up, mat32.DegToRad(delX))
	dx := ctdir.MulQuat(dxq).Sub(ctdir)
	// delY rotates around the right vector
	dyq := mat32.NewQuatAxisAngle(right, mat32.DegToRad(delY))
	dy := ctdir.MulQuat(dyq).Sub(ctdir)

	cm.Pose.Pos = cm.Pose.Pos.Add(dx).Add(dy)
	cm.UpDir.SetMulQuat(dyq) // this is only one that affects up
	cm.CamMu.Unlock()

	cm.LookAtTarget()
}

// Pan moves the camera along the given 2D axes (left/right, up/down),
// relative to current position and orientation (i.e., in the plane of the
// current window view)
// and it moves the target by the same increment, changing the target position.
func (cm *Camera) Pan(delX, delY float32) {
	cm.CamMu.Lock()
	dx := mat32.Vec3{-delX, 0, 0}.MulQuat(cm.Pose.Quat)
	dy := mat32.Vec3{0, -delY, 0}.MulQuat(cm.Pose.Quat)
	td := dx.Add(dy)
	cm.Pose.Pos.SetAdd(td)
	cm.Target.SetAdd(td)
	cm.CamMu.Unlock()
}

// PanAxis moves the camera and target along world X,Y axes
func (cm *Camera) PanAxis(delX, delY float32) {
	cm.CamMu.Lock()
	td := mat32.Vec3{-delX, -delY, 0}
	cm.Pose.Pos.SetAdd(td)
	cm.Target.SetAdd(td)
	cm.CamMu.Unlock()
}

// PanTarget moves the target along world X,Y,Z axes and does LookAt
// at the new target location.  It ensures that the target is not
// identical to the camera position.
func (cm *Camera) PanTarget(delX, delY, delZ float32) {
	td := mat32.Vec3{-delX, -delY, delZ}
	cm.Target.SetAdd(td)
	dist := cm.ViewVector().Length()
	cm.CamMu.Lock()
	if dist == 0 {
		cm.Target.SetAdd(td)
	}
	cm.CamMu.Unlock()
	cm.LookAtTarget()
}

// TargetFmView updates the target location from the current view matrix,
// by projecting the current target distance along the current camera
// view matrix.
func (cm *Camera) TargetFmView() {
	cm.CamMu.Lock()
	trgdist := cm.Pose.Pos.Sub(cm.Target).Length() // distance to existing target
	tpos := mat32.Vec4{0, 0, -trgdist, 1}          // target is that distance along -Z axis in front of me
	cm.Target = mat32.NewVec3FromVec4(tpos.MulMat4(&cm.Pose.Matrix))
	cm.CamMu.Unlock()
}

// Zoom moves along axis given pct closer or further from the target
// it always moves the target back also if it distance is < 1
func (cm *Camera) Zoom(zoomPct float32) {
	ctaxis := cm.ViewVector()
	cm.CamMu.Lock()
	if ctaxis.IsNil() {
		ctaxis.Set(0, 0, 1)
	}
	dist := ctaxis.Length()
	del := ctaxis.MulScalar(zoomPct)
	cm.Pose.Pos.SetAdd(del)
	if zoomPct < 0 && dist < 1 {
		cm.Target.SetAdd(del)
	}
	cm.CamMu.Unlock()
}

// ZoomTo moves along axis in vector pointing through the given 2D point as
// into the camera NDC normalized display coordinates.  Point must be
// 0 normalized, (subtract the Scene ObjBBox.Min) and size of Scene is
// passed as size argument.
// ZoomPct is proportion closer (positive) or further (negative) from the target.
func (cm *Camera) ZoomTo(pt, size image.Point, zoomPct float32) {
	cm.CamMu.Lock()
	fsize := mat32.Vec2{float32(size.X), float32(size.Y)}
	fpt := mat32.Vec2{float32(pt.X), float32(pt.Y)}
	ndc := fpt.WindowToNDC(fsize, mat32.Vec2{}, true) // flipY
	ndc.Z = -1                                        // at closest point
	cdir := mat32.NewVec4FromVec3(ndc, 1).MulMat4(&cm.InvPrjnMatrix)
	cdir.Z = -1
	cdir.W = 0 // vec
	// get world position / transform of camera: matrix is inverse of ViewMatrix
	wdir := mat32.NewVec3FromVec4(cdir.MulMat4(&cm.Pose.Matrix))
	del := wdir.MulScalar(zoomPct)
	cm.Pose.Pos.SetAdd(del)
	cm.CamMu.Unlock()
	cm.UpdateMatrix()
	cm.TargetFmView()
}

// CameraProps define the ToolBar and MenuBar for StructView
var CameraProps = ki.Props{
	"ToolBar": ki.PropSlice{
		{"Defaults", ki.Props{
			"label": "Defaults",
			"icon":  icons.DeviceReset,
		}},
		{"LookAt", ki.Props{
			"icon": icons.Visibility,
			"Args": ki.PropSlice{
				{"Target", ki.BlankProp{}},
				{"UpDir", ki.BlankProp{}},
			},
		}},
		{"Orbit", ki.Props{
			"icon": icons.X3DRotation,
			"Args": ki.PropSlice{
				{"DeltaX", ki.BlankProp{}},
				{"DeltaY", ki.BlankProp{}},
			},
		}},
		{"Pan", ki.Props{
			"icon": icons.PanTool,
			"Args": ki.PropSlice{
				{"DeltaX", ki.BlankProp{}},
				{"DeltaY", ki.BlankProp{}},
			},
		}},
		{"PanAxis", ki.Props{
			"icon": icons.PanTool,
			"Args": ki.PropSlice{
				{"DeltaX", ki.BlankProp{}},
				{"DeltaY", ki.BlankProp{}},
			},
		}},
		{"PanTarget", ki.Props{
			"icon": icons.PanTool,
			"Args": ki.PropSlice{
				{"DeltaX", ki.BlankProp{}},
				{"DeltaY", ki.BlankProp{}},
				{"DeltaZ", ki.BlankProp{}},
			},
		}},
		{"Zoom", ki.Props{
			"icon": icons.ZoomIn,
			"Args": ki.PropSlice{
				{"ZoomPct", ki.BlankProp{}},
			},
		}},
	},
}
