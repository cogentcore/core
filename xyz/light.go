// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"image/color"

	"cogentcore.org/core/mat32"
)

// Light represents a light that illuminates a scene
// these are stored on the Scene object and not within the graph
type Light interface {
	// Name returns name of the light -- lights are accessed by name
	Name() string

	// Color returns color of light
	Color() color.RGBA

	// Lumens returns brightness of light
	Lumens() float32
}

// LightBase provides the base implementation for Light interface
type LightBase struct {

	// name of light -- lights accessed by name so it matters
	Nm string

	// whether light is on or off
	On bool

	// brightness / intensity / strength of the light, in normalized 0-1 units -- just multiplies the color, and is convenient for easily modulating overall brightness
	Lumns float32 `min:"0" step:"0.1"`

	// color of light a full intensity
	Clr color.RGBA
}

// Name returns name of the light -- lights are accessed by name
func (lb *LightBase) Name() string {
	return lb.Nm
}

func (lb *LightBase) Color() color.RGBA {
	return lb.Clr
}

func (lb *LightBase) Lumens() float32 {
	return lb.Lumns
}

/////////////////////////////////////////////////////////////////////////////
//  Light types

// AmbientLight provides diffuse uniform lighting -- typically only one of these
type AmbientLight struct {
	LightBase
}

// NewAmbientLight adds Ambient to given scene, with given name, standard color, and lumens (0-1 normalized)
func NewAmbientLight(sc *Scene, name string, lumens float32, color LightColors) *AmbientLight {
	lt := &AmbientLight{}
	lt.Nm = name
	lt.On = true
	lt.Clr = LightColorMap[color]
	lt.Lumns = lumens
	sc.AddLight(lt)
	return lt
}

// DirLight is directional light, which is assumed to project light toward
// the origin based on its position, with no attenuation, like the Sun.
// For rendering, the position is negated and normalized to get the direction
// vector (i.e., absolute distance doesn't matter)
type DirLight struct {
	LightBase

	// position of direct light -- assumed to point at the origin so this determines direction
	Pos mat32.Vec3
}

// NewDirLight adds direct light to given scene, with given name, standard color, and lumens (0-1 normalized)
// By default it is located overhead and toward the default camera (0, 1, 1) -- change Pos otherwise
func NewDirLight(sc *Scene, name string, lumens float32, color LightColors) *DirLight {
	lt := &DirLight{}
	lt.Nm = name
	lt.On = true
	lt.Clr = LightColorMap[color]
	lt.Lumns = lumens
	lt.Pos.Set(0, 1, 1)
	sc.AddLight(lt)
	return lt
}

// ViewDir gets the direction normal vector, pre-computing the view transform
func (dl *DirLight) ViewDir(viewMat *mat32.Mat4) mat32.Vec3 {
	// adding the 0 in the 4-vector negates any translation factors from the 4 matrix
	return dl.Pos.MulMat4AsVec4(viewMat, 0)
}

// PointLight is an omnidirectional light with a position
// and associated decay factors, which divide the light intensity as a function of
// linear and quadratic distance.  The quadratic factor dominates at longer distances.
type PointLight struct {
	LightBase

	// position of light in world coordinates
	Pos mat32.Vec3

	// Distance linear decay factor -- defaults to .1
	LinDecay float32

	// Distance quadratic decay factor -- defaults to .01 -- dominates at longer distances
	QuadDecay float32
}

// NewPointLight adds point light to given scene, with given name, standard color, and lumens (0-1 normalized)
// By default it is located at 0,5,5 (up and between default camera and origin) -- set Pos to change.
func NewPointLight(sc *Scene, name string, lumens float32, color LightColors) *PointLight {
	lt := &PointLight{}
	lt.Nm = name
	lt.On = true
	lt.Clr = LightColorMap[color]
	lt.Lumns = lumens
	lt.LinDecay = .1
	lt.QuadDecay = .01
	lt.Pos.Set(0, 5, 5)
	sc.AddLight(lt)
	return lt
}

// ViewPos gets the position vector, pre-computing the view transform
func (pl *PointLight) ViewPos(viewMat *mat32.Mat4) mat32.Vec3 {
	return pl.Pos.MulMat4AsVec4(viewMat, 1)
}

// SpotLight is a light with a position and direction and associated decay factors and angles.
// which divide the light intensity as a function of linear and quadratic distance.
// The quadratic factor dominates at longer distances.
type SpotLight struct {
	LightBase
	Pose Pose // position and orientation

	// Angular decay factor -- defaults to 15
	AngDecay float32

	// Cut off angle (in degrees) -- defaults to 45 -- max of 90
	CutoffAngle float32 `max:"90" min:"1"`

	// Distance linear decay factor -- defaults to .01
	LinDecay float32

	// Distance quadratic decay factor -- defaults to .001 -- dominates at longer distances
	QuadDecay float32
}

// NewSpotLight adds spot light to given scene, with given name, standard color, and lumens (0-1 normalized)
// By default it is located at 0,5,5 (up and between default camera and origin) and pointing at the origin.
// Use the Pose LookAt function to point it at other locations.
// In its unrotated state, it points down the -Z axis (i.e., into the scene using default view parameters)
func NewSpotLight(sc *Scene, name string, lumens float32, color LightColors) *SpotLight {
	lt := &SpotLight{}
	lt.Nm = name
	lt.On = true
	lt.Clr = LightColorMap[color]
	lt.Lumns = lumens
	lt.AngDecay = 15
	lt.CutoffAngle = 45
	lt.LinDecay = .01
	lt.QuadDecay = .001
	lt.Pose.Defaults()
	lt.Pose.Pos.Set(0, 2, 5)
	lt.LookAtOrigin()
	sc.AddLight(lt)
	return lt
}

// ViewDir gets the direction normal vector, pre-computing the view transform
func (sl *SpotLight) ViewDir() mat32.Vec3 {
	idmat := mat32.Identity4()
	sl.Pose.UpdateMatrix()
	sl.Pose.UpdateWorldMatrix(idmat)
	// sl.Pose.UpdateMVPMatrix(viewMat, idmat)
	vd := mat32.V3(0, 0, -1).MulMat4AsVec4(&sl.Pose.WorldMatrix, 0).Normal()
	return vd
}

// LookAt points the spotlight at given target location, using given up direction.
func (sl *SpotLight) LookAt(target, upDir mat32.Vec3) {
	sl.Pose.LookAt(target, upDir)
}

// LookAtOrigin points the spotlight at origin with Y axis pointing Up (i.e., standard)
func (sl *SpotLight) LookAtOrigin() {
	sl.LookAt(mat32.Vec3{}, mat32.V3(0, 1, 0))
}

/////////////////////////////////////////////////////////////////////////
//  Scene code

// AddLight adds given light to lights
// see NewX for convenience methods to add specific lights
func (sc *Scene) AddLight(lt Light) {
	sc.Lights.Add(lt.Name(), lt)
}

// ConfigLights configures 3D rendering for current lights
func (sc *Scene) ConfigLights() {
	sc.Phong.ResetNLights()
	for _, ltkv := range sc.Lights.Order {
		lt := ltkv.Value
		clr := mat32.NewVec3Color(lt.Color()).MulScalar(lt.Lumens()).SRGBToLinear()
		switch l := lt.(type) {
		case *AmbientLight:
			sc.Phong.AddAmbientLight(clr)
		case *DirLight:
			sc.Phong.AddDirLight(clr, l.Pos)
		case *PointLight:
			sc.Phong.AddPointLight(clr, l.Pos, l.LinDecay, l.QuadDecay)
		case *SpotLight:
			sc.Phong.AddSpotLight(clr, l.Pose.Pos, l.ViewDir(), l.AngDecay, l.CutoffAngle, l.LinDecay, l.QuadDecay)
		}
	}
}

/////////////////////////////////////////////////////////////////////////\
//  Standard Light Colors

// http://planetpixelemporium.com/tutorialpages/light.html

// LightColors are standard light colors for different light sources
type LightColors int32 //enums:enum

const (
	DirectSun LightColors = iota
	CarbonArc
	Halogen
	Tungsten100W
	Tungsten40W
	Candle
	Overcast
	FluorWarm
	FluorStd
	FluorCool
	FluorFull
	FluorGrow
	MercuryVapor
	SodiumVapor
	MetalHalide
)

// LightColorMap provides a map of named light colors
var LightColorMap = map[LightColors]color.RGBA{
	DirectSun:    {255, 255, 255, 255},
	CarbonArc:    {255, 250, 244, 255},
	Halogen:      {255, 241, 224, 255},
	Tungsten100W: {255, 214, 170, 255},
	Tungsten40W:  {255, 197, 143, 255},
	Candle:       {255, 147, 41, 255},
	Overcast:     {201, 226, 255, 255},
	FluorWarm:    {255, 244, 229, 255},
	FluorStd:     {244, 255, 250, 255},
	FluorCool:    {212, 235, 255, 255},
	FluorFull:    {255, 244, 242, 255},
	FluorGrow:    {255, 239, 247, 255},
	MercuryVapor: {216, 247, 255, 255},
	SodiumVapor:  {255, 209, 178, 255},
	MetalHalide:  {242, 252, 255, 255},
}
