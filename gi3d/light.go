// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/ki/kit"
)

// http://planetpixelemporium.com/tutorialpages/light.html

// LightColors are standard light colors for different light sources
type LightColors int

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
	LightColorsN
)

//go:generate stringer -type=LightColors

var KiT_LightColors = kit.Enums.AddEnum(LightColorsN, false, nil)

// LightColorMap provides a map of named light colors
var LightColorMap = map[LightColors]gi.Color{
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

// Light represents a light that illuminates a scene
// these are stored on the overall scene object and not within the graph
type Light interface {
	// Name returns name of the light -- lights are accessed by name
	Name() string

	// Color returns color of light
	Color() gi.Color

	// Lumens returns brightness of light
	Lumens() float32
}

// LightBase provides the base implementation for Light interface
type LightBase struct {
	Nm    string   `desc:"name of light -- lights accessed by name so it matters"`
	On    bool     `desc:"whether light is on or off"`
	Lumns float32  `desc:"brightness / intensity / strength of the light, in normalized 0-1 units -- just multiplies the color, and is convenient for easily modulating overall brightness"`
	Clr   gi.Color `desc:"color of light a full intensity"`
}

var KiT_LightBase = kit.Types.AddType(&LightBase{}, nil)

// Name returns name of the light -- lights are accessed by name
func (lb *LightBase) Name() string {
	return lb.Nm
}

func (lb *LightBase) Color() gi.Color {
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

var KiT_AmbientLight = kit.Types.AddType(&AmbientLight{}, nil)

// AddNewAmbientLight adds Ambient to given scene, with given name, standard color, and lumens (0-1 normalized)
func AddNewAmbientLight(sc *Scene, name string, lumens float32, color LightColors) *AmbientLight {
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
	Pos mat32.Vec3 `desc:"position of direct light -- assumed to point at the origin so this determines direction"`
}

var KiT_DirLight = kit.Types.AddType(&DirLight{}, nil)

// AddNewDirLight adds direct light to given scene, with given name, standard color, and lumens (0-1 normalized)
// By default it is located overhead and toward the default camera (0, 1, 1) -- change Pos otherwise
func AddNewDirLight(sc *Scene, name string, lumens float32, color LightColors) *DirLight {
	lt := &DirLight{}
	lt.Nm = name
	lt.On = true
	lt.Clr = LightColorMap[color]
	lt.Lumns = lumens
	lt.Pos.Set(0, 1, 1)
	sc.AddLight(lt)
	return lt
}

// Dir gets the direction normal vector, pre-computing the view transform
func (dl *DirLight) Dir(viewMat *mat32.Mat4) mat32.Vec3 {
	dir4 := mat32.NewVec4FromVec3(dl.Pos, 0).MulMat4(viewMat)
	return mat32.NewVec3FromVec4(dir4).Normal()
}

// PointLight is an omnidirectional light with a position
// and associated decay factors
type PointLight struct {
	LightBase
	Pos       mat32.Vec3 // position of light
	LinDecay  float32    // Distance linear decay factor
	QuadDecay float32    // Distance quadratic decay factor
}

var KiT_PointLight = kit.Types.AddType(&PointLight{}, nil)

// AddNewPointLight adds point light to given scene, with given name, standard color, and lumens (0-1 normalized)
// By default it is located near the default camera position -- change Pos otherwise
func AddNewPointLight(sc *Scene, name string, lumens float32, color LightColors) *PointLight {
	lt := &PointLight{}
	lt.Nm = name
	lt.On = true
	lt.Clr = LightColorMap[color]
	lt.Lumns = lumens
	lt.LinDecay = 1
	lt.QuadDecay = 1
	lt.Pos.Set(0, 5, 5)
	sc.AddLight(lt)
	return lt
}

// Dir gets the direction normal vector, pre-computing the view transform
func (dl *PointLight) Dir(viewMat *mat32.Mat4) mat32.Vec3 {
	dir4 := mat32.NewVec4FromVec3(dl.Pos, 0).MulMat4(viewMat)
	return mat32.NewVec3FromVec4(dir4).Normal()
}

// Spotlight is a light with a position and direction
// and associated decay factors and angles
type SpotLight struct {
	LightBase
	Pose           Pose    // position and orientation
	LinearDecay    float32 // Distance linear decay factor
	QuadraticDecay float32 // Distance quadratic decay factor
	AngularDecay   float32 // Angular decay factor
	CutoffAngle    float32 // Cut off angle (in radians?)
}

var KiT_SpotLight = kit.Types.AddType(&SpotLight{}, nil)

// SetLightsUnis sets the lights and recompiles the programs accordingly
// Must be called with proper context activated, on main thread
func (rn *Renderers) SetLightsUnis(sc *Scene) {
	lu, ok := rn.Unis["Lights"]
	if !ok {
		return
	}
	var ambs []mat32.Vec3
	var dirs []mat32.Vec3
	// var points []mat32.Vec3
	// var spots []mat32.Vec3
	for _, lt := range sc.Lights {
		clr := ColorToVec3f(lt.Color()).MulScalar(lt.Lumens())
		switch l := lt.(type) {
		case *AmbientLight:
			ambs = append(ambs, clr)
		case *DirLight:
			dirs = append(dirs, clr)
			dirs = append(dirs, l.Dir(&sc.Camera.ViewMatrix))
		}
	}

	// set new lengths first
	ambu := lu.UniformByName("AmbLights")
	ambu.SetLen(len(ambs))
	diru := lu.UniformByName("DirLights")
	diru.SetLen(len(dirs))

	lu.Resize()

	if len(ambs) > 0 {
		ambu.SetValue(ambs)
	}
	if len(dirs) > 0 {
		diru.SetValue(dirs)
	}
}
