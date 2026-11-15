// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"image"
	"testing"

	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

// assertVec asserts that two vectors are near each other.
func assertVec(t *testing.T, want, got math32.Vector3, msg string) {
	t.Helper()
	assert.InDelta(t, 0, want.Sub(got).Length(), 1e-4, "%s: want %v, got %v", msg, want, got)
}

// TestPoseRotations tests setting and changing the rotation of a pose.
func TestPoseRotations(t *testing.T) {
	var ps Pose
	ps.Defaults()
	ps.SetEulerRotation(0, 45, 0)
	assertVec(t, math32.Vec3(0, 45, 0), ps.EulerRotation(), "euler degrees")
	assertVec(t, math32.Vec3(0, math32.Pi/4, 0), ps.EulerRotationRad(), "euler radians")
	ps.SetEulerRotation(0, 90, 0)
	assertVec(t, math32.Vec3(0, 0, -1), ps.Quat.MulVector(math32.Vec3(1, 0, 0)), "90 around y turns x to -z")

	ps.SetEulerRotationRad(0, 0, math32.Pi/2)
	assertVec(t, math32.Vec3(0, 1, 0), ps.Quat.MulVector(math32.Vec3(1, 0, 0)), "90 around z turns x to y")

	ps.SetAxisRotation(0, 0, 1, 90)
	ps.RotateOnAxis(0, 0, 1, 90)
	assertVec(t, math32.Vec3(-1, 0, 0), ps.Quat.MulVector(math32.Vec3(1, 0, 0)), "two rotations of 90")
	ps.SetAxisRotationRad(0, 1, 0, 0)
	ps.RotateOnAxisRad(1, 0, 0, math32.Pi)
	assertVec(t, math32.Vec3(0, -1, 0), ps.Quat.MulVector(math32.Vec3(0, 1, 0)), "180 around x")
	ps.Quat.SetIdentity()
	ps.RotateEuler(90, 0, 0)
	assertVec(t, math32.Vec3(0, 0, 1), ps.Quat.MulVector(math32.Vec3(0, 1, 0)), "euler 90 around x turns y to z")
	ps.Quat.SetIdentity()
	ps.RotateEulerRad(0, math32.Pi, 0, 0)
	assertVec(t, math32.Vec3(-1, 0, 0), ps.Quat.MulVector(math32.Vec3(1, 0, 0)), "euler radians")
}

// TestPoseMoves tests moving a pose on axes, absolute and relative to
// its rotation.
func TestPoseMoves(t *testing.T) {
	var ps Pose
	ps.Defaults()
	ps.SetAxisRotation(0, 1, 0, 90)
	ps.MoveOnAxis(1, 0, 0, 2)
	assertVec(t, math32.Vec3(0, 0, -2), ps.Pos, "relative to the rotation")
	ps.MoveOnAxisAbs(0, 3, 0, 1)
	assertVec(t, math32.Vec3(0, 1, -2), ps.Pos, "absolute, normalized")
}

// TestPoseMatrices tests the matrices of a pose, and its world values
// within a parent.
func TestPoseMatrices(t *testing.T) {
	var ps Pose
	ps.Pos.Set(1, 2, 3)
	ps.Scale.Set(2, 2, 2)
	ps.SetAxisRotation(0, 1, 0, 90)
	ps.UpdateMatrix()
	m := ps.Matrix

	var cp Pose
	cp.SetMatrix(&m)
	assertVec(t, ps.Pos, cp.Pos, "pos from matrix")
	assertVec(t, ps.Scale, cp.Scale, "scale from matrix")
	assert.Contains(t, ps.String(), "Pos: ")
	assert.Contains(t, ps.GenGoSet("p"), "p.Pos.Set(")

	var par math32.Matrix4
	par.SetTranslation(10, 0, 0)
	ps.UpdateWorldMatrix(&par)
	assertVec(t, math32.Vec3(11, 2, 3), ps.WorldPos(), "world pos")
	assertVec(t, math32.Vec3(2, 2, 2), ps.WorldScale(), "world scale")
	wq := ps.WorldQuat()
	assertVec(t, math32.Vec3(0, 0, -1), wq.MulVector(math32.Vec3(1, 0, 0)), "world rotation")

	// the world rotation includes the rotation of the parent
	var rpar math32.Matrix4
	rpar.SetRotationY(math32.Pi / 4)
	var child Pose
	child.Defaults()
	child.UpdateMatrix()
	child.UpdateWorldMatrix(&rpar)
	assert.InDelta(t, math32.Pi/4, child.WorldEulerRotation().Y, 1e-4, "the world rotation, not the local one")

	var mp Pose
	mp.Defaults()
	mp.UpdateMatrix()
	var tr math32.Matrix4
	tr.SetTranslation(0, 5, 0)
	mp.MulMatrix(&tr)
	assertVec(t, math32.Vec3(0, 5, 0), mp.Pos, "multiplied by a translation")

	var cpy Pose
	cpy.CopyFrom(&ps)
	assert.Equal(t, ps.Pos, cpy.Pos)
	assert.Equal(t, ps.Matrix, cpy.Matrix)
}

// newTestCamera returns a camera with defaults, looking at the origin
// from (0, 0, 10).
func newTestCamera() *Camera {
	cm := &Camera{}
	cm.Defaults()
	cm.SetAspect(image.Pt(400, 300))
	cm.UpdateMatrix()
	return cm
}

// TestCameraLook tests pointing a camera and its view vector.
func TestCameraLook(t *testing.T) {
	cm := newTestCamera()
	assertVec(t, math32.Vec3(0, 0, 10), cm.Pose.Pos, "default position")
	assertVec(t, math32.Vec3(0, 0, 10), cm.ViewVector(), "view vector")
	assert.InDelta(t, 10, cm.DistanceTo(math32.Vector3{}), 1e-5)
	dim, sign := cm.ViewMainAxis()
	assert.Equal(t, math32.Z, dim)
	assert.Equal(t, float32(1), sign)
	assert.InDelta(t, 4.0/3, cm.Aspect, 1e-5)

	cm.Pose.Pos.Set(-20, 1, 0)
	cm.LookAt(math32.Vec3(0, 1, 0), math32.Vector3{}) // default up
	assert.Equal(t, math32.Vec3(0, 1, 0), cm.UpDir)
	dim, sign = cm.ViewMainAxis()
	assert.Equal(t, math32.X, dim)
	assert.Equal(t, float32(-1), sign)
	assertVec(t, math32.Vec3(1, 0, 0), cm.Pose.Quat.MulVector(math32.Vec3(0, 0, -1)), "looks along +x")

	cm.Pose.Pos.Set(0, 30, 1)
	cm.LookAtOrigin()
	dim, _ = cm.ViewMainAxis()
	assert.Equal(t, math32.Y, dim)
	cm.Target.Set(0, 0, 0)
	cm.LookAtTarget()
	assert.Contains(t, cm.GenGoSet("c"), "c.Pose")
}

// TestCameraMoves tests orbiting, panning, and zooming a camera.
func TestCameraMoves(t *testing.T) {
	cm := newTestCamera()
	cm.Orbit(90, 0)
	assert.InDelta(t, 10, cm.DistanceTo(cm.Target), 1e-4, "orbit keeps the distance")
	assert.InDelta(t, 10, math32.Abs(cm.Pose.Pos.X), 1e-3, "orbit 90 moves to the x axis")
	assertVec(t, cm.Target.Sub(cm.Pose.Pos).Normal(), cm.Pose.Quat.MulVector(math32.Vec3(0, 0, -1)), "still looking at the target")

	cm = newTestCamera()
	cm.Orbit(0, 30)
	assert.InDelta(t, 10, cm.DistanceTo(cm.Target), 1e-4)
	assert.Greater(t, math32.Abs(cm.Pose.Pos.Y), float32(4))

	cm = newTestCamera()
	cm.Pan(2, 1)
	assertVec(t, math32.Vec3(-2, -1, 10), cm.Pose.Pos, "pan moves the camera")
	assertVec(t, math32.Vec3(-2, -1, 0), cm.Target, "and the target")
	cm.PanAxis(1, 1)
	assertVec(t, math32.Vec3(-3, -2, 0), cm.Target, "pan axis")
	cm.PanTarget(-3, -2, 1)
	assertVec(t, math32.Vec3(0, 0, 1), cm.Target, "pan target")

	cm = newTestCamera()
	cm.Zoom(-0.5)
	assertVec(t, math32.Vec3(0, 0, 5), cm.Pose.Pos, "zoom in by half")
	cm.Zoom(1)
	assertVec(t, math32.Vec3(0, 0, 10), cm.Pose.Pos, "zoom out")

	cm = newTestCamera()
	cm.ZoomTo(image.Pt(200, 150), image.Pt(400, 300), 2)
	assert.InDelta(t, 0, cm.Pose.Pos.X, 1e-4, "zoom to the center moves along the view")
	assert.Less(t, cm.Pose.Pos.Z, float32(10), "closer")
	assertVec(t, math32.Vec3(0, 0, 0), math32.Vec3(cm.Target.X, cm.Target.Y, 0), "the target is on the view axis")

	cm = newTestCamera()
	cm.Pose.SetAxisRotation(0, 1, 0, 90) // looking along -x
	cm.UpdateMatrix()
	cm.TargetFromView()
	assertVec(t, math32.Vec3(-10, 0, 10), cm.Target, "the target at the same distance in front")
}
