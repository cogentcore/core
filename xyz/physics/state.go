// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package physics

import (
	"math"

	"cogentcore.org/core/math32"
)

// State contains the basic physical state including position, orientation, velocity.
// These are only the values that can be either relative or absolute -- other physical
// state values such as Mass should go in Rigid.
type State struct {

	// position of center of mass of object
	Pos math32.Vector3

	// rotation specified as a Quat
	Quat math32.Quat

	// linear velocity
	LinVel math32.Vector3

	// angular velocity
	AngVel math32.Vector3
}

// Defaults sets defaults only if current values are nil
func (ps *State) Defaults() {
	if ps.Quat.IsNil() {
		ps.Quat.SetIdentity()
	}
}

//////// 	State updates

// FromRel sets state from relative values compared to a parent state
func (ps *State) FromRel(rel, par *State) {
	ps.Quat = rel.Quat.Mul(par.Quat)
	ps.Pos = rel.Pos.MulQuat(par.Quat).Add(par.Pos)
	ps.LinVel = rel.LinVel.MulQuat(rel.Quat).Add(par.LinVel)
	ps.AngVel = rel.AngVel.MulQuat(rel.Quat).Add(par.AngVel)
}

// AngMotionMax is maximum angular motion that can be taken per update
const AngMotionMax = math.Pi / 4

// StepByAngVel steps the Quat rotation from angular velocity
func (ps *State) StepByAngVel(step float32) {
	ang := math32.Sqrt(ps.AngVel.Dot(ps.AngVel))

	// limit the angular motion
	if ang*step > AngMotionMax {
		ang = AngMotionMax / step
	}
	var axis math32.Vector3
	if ang < 0.001 {
		// use Taylor's expansions of sync function
		axis = ps.AngVel.MulScalar(0.5*step - (step*step*step)*0.020833333333*ang*ang)
	} else {
		// sync(fAngle) = sin(c*fAngle)/t
		axis = ps.AngVel.MulScalar(math32.Sin(0.5*ang*step) / ang)
	}
	var dq math32.Quat
	dq.SetFromAxisAngle(axis, ang*step)
	ps.Quat = dq.Mul(ps.Quat)
	ps.Quat.Normalize()
}

// StepByLinVel steps the Pos from the linear velocity
func (ps *State) StepByLinVel(step float32) {
	ps.Pos = ps.Pos.Add(ps.LinVel.MulScalar(step))
}

//////// 		Moving

// Move moves (translates) Pos by given amount, and sets the LinVel to the given
// delta -- this can be useful for Scripted motion to track movement.
func (ps *State) Move(delta math32.Vector3) {
	ps.LinVel = delta
	ps.Pos.SetAdd(delta)
}

// MoveOnAxis moves (translates) the specified distance on the specified local axis,
// relative to the current rotation orientation.
// The axis is normalized prior to aplying the distance factor.
// Sets the LinVel to motion vector.
func (ps *State) MoveOnAxis(x, y, z, dist float32) { //types:add
	ps.LinVel = math32.Vec3(x, y, z).Normal().MulQuat(ps.Quat).MulScalar(dist)
	ps.Pos.SetAdd(ps.LinVel)
}

// MoveOnAxisAbs moves (translates) the specified distance on the specified local axis,
// in absolute X,Y,Z coordinates (does not apply the Quat rotation factor.
// The axis is normalized prior to aplying the distance factor.
// Sets the LinVel to motion vector.
func (ps *State) MoveOnAxisAbs(x, y, z, dist float32) { //types:add
	ps.LinVel = math32.Vec3(x, y, z).Normal().MulScalar(dist)
	ps.Pos.SetAdd(ps.LinVel)
}

//////// 		Rotating

// SetEulerRotation sets the rotation in Euler angles (degrees).
func (ps *State) SetEulerRotation(x, y, z float32) { //types:add
	ps.Quat.SetFromEuler(math32.Vec3(x, y, z).MulScalar(math32.DegToRadFactor))
}

// SetEulerRotationRad sets the rotation in Euler angles (radians).
func (ps *State) SetEulerRotationRad(x, y, z float32) {
	ps.Quat.SetFromEuler(math32.Vec3(x, y, z))
}

// EulerRotation returns the current rotation in Euler angles (degrees).
func (ps *State) EulerRotation() math32.Vector3 { //types:add
	return ps.Quat.ToEuler().MulScalar(math32.RadToDegFactor)
}

// EulerRotationRad returns the current rotation in Euler angles (radians).
func (ps *State) EulerRotationRad() math32.Vector3 {
	return ps.Quat.ToEuler()
}

// SetAxisRotation sets rotation from local axis and angle in degrees.
func (ps *State) SetAxisRotation(x, y, z, angle float32) { //types:add
	ps.Quat.SetFromAxisAngle(math32.Vec3(x, y, z), math32.DegToRad(angle))
}

// SetAxisRotationRad sets rotation from local axis and angle in radians.
func (ps *State) SetAxisRotationRad(x, y, z, angle float32) {
	ps.Quat.SetFromAxisAngle(math32.Vec3(x, y, z), angle)
}

// RotateOnAxis rotates around the specified local axis the specified angle in degrees.
func (ps *State) RotateOnAxis(x, y, z, angle float32) { //types:add
	ps.Quat.SetMul(math32.NewQuatAxisAngle(math32.Vec3(x, y, z), math32.DegToRad(angle)))
}

// RotateOnAxisRad rotates around the specified local axis the specified angle in radians.
func (ps *State) RotateOnAxisRad(x, y, z, angle float32) {
	ps.Quat.SetMul(math32.NewQuatAxisAngle(math32.Vec3(x, y, z), angle))
}

// RotateEuler rotates by given Euler angles (in degrees) relative to existing rotation.
func (ps *State) RotateEuler(x, y, z float32) { //types:add
	ps.Quat.SetMul(math32.NewQuatEuler(math32.Vec3(x, y, z).MulScalar(math32.DegToRadFactor)))
}

// RotateEulerRad rotates by given Euler angles (in radians) relative to existing rotation.
func (ps *State) RotateEulerRad(x, y, z, angle float32) {
	ps.Quat.SetMul(math32.NewQuatEuler(math32.Vec3(x, y, z)))
}
