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
	Dir   mat32.Vec3 `desc:"direction of light vectdor"`
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
	_, nl, _ := vars.ValByIdxTry(int(NLightsSet), "NLights", 0)
	nl.CopyBytes(unsafe.Pointer(&ph.NLights))

	for i := 0; i < int(ph.NLights.Ambient); i++ {
		_, al, _ := vars.ValByIdxTry(int(LightsSet), "AmbLights", i)
		al.CopyBytes(unsafe.Pointer(&ph.Ambient[i]))
	}

	for i := 0; i < int(ph.NLights.Dir); i++ {
		_, dl, _ := vars.ValByIdxTry(int(LightsSet), "DirLights", i)
		dl.CopyBytes(unsafe.Pointer(&ph.Dir[i]))
	}

	for i := 0; i < int(ph.NLights.Point); i++ {
		_, pl, _ := vars.ValByIdxTry(int(LightsSet), "PointLights", i)
		pl.CopyBytes(unsafe.Pointer(&ph.Point[i]))
	}

	for i := 0; i < int(ph.NLights.Spot); i++ {
		_, sl, _ := vars.ValByIdxTry(int(LightsSet), "SpotLights", i)
		sl.CopyBytes(unsafe.Pointer(&ph.Spot[i]))
	}
	ph.Sys.Mem.SyncToGPU()
}

// AddAmbientLight adds Ambient light at given position
func (ph *Phong) AddAmbientLight(color mat32.Vec3) {
	ph.Ambient[ph.NLights.Ambient].Color = color
	ph.NLights.Ambient++
}

// AddDirLight adds directional light
func (ph *Phong) AddDirLight(color, dir mat32.Vec3) {
	ph.Dir[ph.NLights.Dir].Color = color
	ph.Dir[ph.NLights.Dir].Dir = dir
	ph.NLights.Dir++
}

// AddPointLight adds point light
func (ph *Phong) AddPointLight(color, pos mat32.Vec3, linDecay, quadDecay float32) {
	ph.Point[ph.NLights.Point].Color = color
	ph.Point[ph.NLights.Point].Pos = pos
	ph.Point[ph.NLights.Point].Decay = mat32.Vec3{X: linDecay, Y: quadDecay}
	ph.NLights.Point++
}

// AddSpotLight adds point light
func (ph *Phong) AddSpotLight(color, pos, dir mat32.Vec3, angDecay, cutAngle, linDecay, quadDecay float32) {
	ph.Spot[ph.NLights.Spot].Color = color
	ph.Spot[ph.NLights.Spot].Pos = pos
	ph.Spot[ph.NLights.Spot].Dir = dir
	ph.Spot[ph.NLights.Spot].Decay = mat32.Vec4{X: angDecay, Y: cutAngle, Z: linDecay, W: quadDecay}
	ph.NLights.Spot++
}
