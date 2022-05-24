// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"unsafe"

	"github.com/goki/mat32"
)

// Number of different lights active
type NLights struct {
	Ambient int32
	Dir     int32
	Point   int32
	Spot    int32
}

func (nl *NLights) Reset() {
	nl.Ambient = 0
	nl.Dir = 0
	nl.Point = 0
	nl.Spot = 0
}

// AmbientLight provides diffuse uniform lighting -- typically only one of these
type AmbientLight struct {
	Color mat32.Vec3 `desc:"color of light -- multiplies ambient color of materials"`
	pad0  float32
}

// DirLight is directional light, which is assumed to project light toward
// the origin based on its position, with no attenuation, like the Sun.
// For rendering, the position is negated and normalized to get the direction
// vector (i.e., absolute distance doesn't matter)
type DirLight struct {
	Color mat32.Vec3 `desc:"color of light at full intensity"`
	pad0  float32
	Pos   mat32.Vec3 `desc:"position of light vector -- think of it shining down from this position toward the origin, i.e., the negation of this position is the vector."`
	pad1  float32
}

// PointLight is an omnidirectional light with a position
// and associated decay factors, which divide the light intensity as a function of
// linear and quadratic distance.  The quadratic factor dominates at longer distances.
type PointLight struct {
	Color mat32.Vec3 `desc:"color of light a full intensity"`
	pad0  float32
	Pos   mat32.Vec3 `desc:"position of light in world coordinates"`
	pad1  float32
	Decay mat32.Vec3 `desc:"X = Linear, Y = Quad: Distance linear decay factor -- defaults to .1; Distance quadratic decay factor -- defaults to .01 -- dominates at longer distances"`
	pad2  float32
}

// Spotlight is a light with a position and direction and
// associated decay factors and angles,
// which divide the light intensity as a function of
// linear and quadratic distance.
// The quadratic factor dominates at longer distances.
type SpotLight struct {
	Color mat32.Vec3 `desc:"color of light a full intensity"`
	pad0  float32
	Pos   mat32.Vec3 `desc:"position of light in world coordinates"`
	pad1  float32
	Dir   mat32.Vec3 `desc:"direction of light vector"`
	pad2  float32
	Decay mat32.Vec4 `desc:"X = Angular Decay, Y = CutAngle, Z = LinDecay, W = QuadDecay: Angular decay factor -- defaults to 15; Cut off angle (in degrees) -- defaults to 45 -- max of 90; Distance linear decay factor -- defaults to 1; Distance quadratic decay factor -- defaults to 1 -- dominates at longer distances"`
}

// ConfigLights configures the rendering for the lights that have been added.
func (ph *Phong) ConfigLights() {
	vars := ph.Sys.Vars()
	vs := vars.SetMap[int(NLightSet)]
	nlv, nl, _ := vs.ValByIdxTry("NLights", 0)
	nl.CopyBytes(unsafe.Pointer(&ph.NLights))
	vs.BindDynVal(vars, nlv, nl)

	vs = vars.SetMap[int(LightSet)]
	alv, al, _ := vs.ValByIdxTry("AmbLights", 0)
	al.CopyBytes(unsafe.Pointer(&ph.Ambient[0]))
	vs.BindDynVal(vars, alv, al)

	dlv, dl, _ := vs.ValByIdxTry("DirLights", 0)
	dl.CopyBytes(unsafe.Pointer(&ph.Dir[0]))
	vs.BindDynVal(vars, dlv, dl)

	plv, pl, _ := vs.ValByIdxTry("PointLights", 0)
	pl.CopyBytes(unsafe.Pointer(&ph.Point[0]))
	vs.BindDynVal(vars, plv, pl)

	slv, sl, _ := vs.ValByIdxTry("SpotLights", 0)
	sl.CopyBytes(unsafe.Pointer(&ph.Spot[0]))
	vs.BindDynVal(vars, slv, sl)
}

// ResetNLights resets number of lights to 0 -- for reconfig
func (ph *Phong) ResetNLights() {
	ph.NLights.Reset()
}

// AddAmbientLight adds Ambient light at given position
func (ph *Phong) AddAmbientLight(color mat32.Vec3) {
	ph.SetAmbientLight(int(ph.NLights.Ambient), color)
	ph.NLights.Ambient++
}

// SetAmbientLight sets Ambient light at index to given position
func (ph *Phong) SetAmbientLight(idx int, color mat32.Vec3) {
	ph.Ambient[idx].Color = color
}

// AddDirLight adds directional light
func (ph *Phong) AddDirLight(color, pos mat32.Vec3) {
	ph.SetDirLight(int(ph.NLights.Dir), color, pos)
	ph.NLights.Dir++
}

// SetDirLight sets directional light at given index
func (ph *Phong) SetDirLight(idx int, color, pos mat32.Vec3) {
	ph.Dir[idx].Color = color
	ph.Dir[idx].Pos = pos
}

// AddPointLight adds point light.
// Defaults: linDecay=.1, quadDecay=.01
func (ph *Phong) AddPointLight(color, pos mat32.Vec3, linDecay, quadDecay float32) {
	ph.SetPointLight(int(ph.NLights.Point), color, pos, linDecay, quadDecay)
	ph.NLights.Point++
}

// SetPointLight sets point light at given index.
// Defaults: linDecay=.1, quadDecay=.01
func (ph *Phong) SetPointLight(idx int, color, pos mat32.Vec3, linDecay, quadDecay float32) {
	ph.Point[idx].Color = color
	ph.Point[idx].Pos = pos
	ph.Point[idx].Decay = mat32.Vec3{X: linDecay, Y: quadDecay}
}

// AddSpotLight adds spot light
// Defaults: angDecay=15, cutAngle=45 (max 90), linDecay=.01, quadDecay=0.001
func (ph *Phong) AddSpotLight(color, pos, dir mat32.Vec3, angDecay, cutAngle, linDecay, quadDecay float32) {
	ph.SetSpotLight(int(ph.NLights.Spot), color, pos, dir, angDecay, cutAngle, linDecay, quadDecay)
	ph.NLights.Spot++
}

// SetSpotLight sets spot light at given index
// Defaults: angDecay=15, cutAngle=45 (max 90), linDecay=.01, quadDecay=0.001
func (ph *Phong) SetSpotLight(idx int, color, pos, dir mat32.Vec3, angDecay, cutAngle, linDecay, quadDecay float32) {
	ph.Spot[idx].Color = color
	ph.Spot[idx].Pos = pos
	ph.Spot[idx].Dir = dir
	ph.Spot[idx].Decay = mat32.Vec4{X: angDecay, Y: cutAngle, Z: linDecay, W: quadDecay}
}
