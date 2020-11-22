// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/gist"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Light represents a light that illuminates a scene
// these are stored on the overall scene and not within the graph
type Light interface {
	// Name returns name of the light -- lights are accessed by name
	Name() string

	// Color returns color of light
	Color() gist.Color

	// Lumens returns brightness of light
	Lumens() float32
}

// LightBase provides the base implementation for Light interface
type LightBase struct {
	Nm    string     `desc:"name of light -- lights accessed by name so it matters"`
	On    bool       `desc:"whether light is on or off"`
	Lumns float32    `min:"0" step:"0.1" desc:"brightness / intensity / strength of the light, in normalized 0-1 units -- just multiplies the color, and is convenient for easily modulating overall brightness"`
	Clr   gist.Color `desc:"color of light a full intensity"`
}

var KiT_LightBase = kit.Types.AddType(&LightBase{}, nil)

// Name returns name of the light -- lights are accessed by name
func (lb *LightBase) Name() string {
	return lb.Nm
}

func (lb *LightBase) Color() gist.Color {
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
	Pos       mat32.Vec3 `desc:"position of light in world coordinates"`
	LinDecay  float32    `desc:"Distance linear decay factor -- defaults to .1"`
	QuadDecay float32    `desc:"Distance quadratic decay factor -- defaults to .01 -- this is "`
}

var KiT_PointLight = kit.Types.AddType(&PointLight{}, nil)

// AddNewPointLight adds point light to given scene, with given name, standard color, and lumens (0-1 normalized)
// By default it is located at 0,5,5 (up and between default camera and origin) -- set Pos to change.
func AddNewPointLight(sc *Scene, name string, lumens float32, color LightColors) *PointLight {
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

// Spotlight is a light with a position and direction and associated decay factors and angles.
// which divide the light intensity as a function of linear and quadratic distance.
// The quadratic factor dominates at longer distances.
type SpotLight struct {
	LightBase
	Pose        Pose    // position and orientation
	AngDecay    float32 `desc:"Angular decay factor -- defaults to 15"`
	CutoffAngle float32 `max:"90" min:"1" desc:"Cut off angle (in degrees) -- defaults to 45 -- max of 90"`
	LinDecay    float32 `desc:"Distance linear decay factor -- defaults to 1"`
	QuadDecay   float32 `desc:"Distance quadratic decay factor -- defaults to 1"`
}

var KiT_SpotLight = kit.Types.AddType(&SpotLight{}, nil)

// AddNewSpotLight adds spot light to given scene, with given name, standard color, and lumens (0-1 normalized)
// By default it is located at 0,5,5 (up and between default camera and origin) and pointing at the origin.
// Use the Pose LookAt function to point it at other locations.
// In its unrotated state, it points down the -Z axis (i.e., into the scene using default view parameters)
func AddNewSpotLight(sc *Scene, name string, lumens float32, color LightColors) *SpotLight {
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

// ViewPos gets the position of the light, pre-computing the view transform
func (sl *SpotLight) ViewPos(viewMat *mat32.Mat4) mat32.Vec3 {
	return sl.Pose.Pos.MulMat4AsVec4(viewMat, 1) // note: 1 and no Normal
}

// ViewDir gets the direction normal vector, pre-computing the view transform
func (sl *SpotLight) ViewDir(viewMat *mat32.Mat4) mat32.Vec3 {
	idmat := mat32.NewMat4()
	sl.Pose.UpdateMatrix()
	sl.Pose.UpdateWorldMatrix(idmat)
	sl.Pose.UpdateMVPMatrix(viewMat, idmat)
	vd := mat32.Vec3{0, 0, -1}.MulMat4AsVec4(&sl.Pose.MVMatrix, 0).Normal()
	return vd
}

// LookAt points the spotlight at given target location, using given up direction.
func (sl *SpotLight) LookAt(target, upDir mat32.Vec3) {
	sl.Pose.LookAt(target, upDir)
}

// LookAtOrigin points the spotlight at origin with Y axis pointing Up (i.e., standard)
func (sl *SpotLight) LookAtOrigin() {
	sl.LookAt(mat32.Vec3Zero, mat32.Vec3Y)
}

/////////////////////////////////////////////////////////////////////////\
//  Set Lights to Renderers

// SetLightsUnis sets the lights and recompiles the programs accordingly
// Must be called with proper context activated, on main thread
func (rn *Renderers) SetLightsUnis(sc *Scene) {
	lu, ok := rn.Unis["Lights"]
	if !ok {
		return
	}
	var ambs []mat32.Vec3
	var dirs []mat32.Vec3
	var points []mat32.Vec3
	var spots []mat32.Vec3
	sc.Camera.CamMu.RLock()
	for _, lt := range sc.Lights {
		clr := ColorToVec3f(lt.Color()).MulScalar(lt.Lumens())
		switch l := lt.(type) {
		case *AmbientLight:
			ambs = append(ambs, clr)
		case *DirLight:
			dirs = append(dirs, clr)
			dirs = append(dirs, l.ViewDir(&sc.Camera.ViewMatrix))
		case *PointLight:
			points = append(points, clr)
			points = append(points, l.ViewPos(&sc.Camera.ViewMatrix))
			points = append(points, mat32.Vec3{l.LinDecay, l.QuadDecay, 0})
		case *SpotLight:
			spots = append(spots, clr)
			spots = append(spots, l.ViewPos(&sc.Camera.ViewMatrix))
			spots = append(spots, l.ViewDir(&sc.Camera.ViewMatrix))
			spots = append(spots, mat32.Vec3{l.AngDecay, l.CutoffAngle, l.LinDecay})
			spots = append(spots, mat32.Vec3{l.QuadDecay, 0, 0})
		}
	}
	sc.Camera.CamMu.RUnlock()
	// set new lengths first
	ambu := lu.UniformByName("AmbLights")
	ambu.SetLen(len(ambs))
	diru := lu.UniformByName("DirLights")
	diru.SetLen(len(dirs))
	ptu := lu.UniformByName("PointLights")
	ptu.SetLen(len(points))
	spu := lu.UniformByName("SpotLights")
	spu.SetLen(len(spots))

	lu.Resize()

	if len(ambs) > 0 {
		ambu.SetValue(ambs)
	}
	if len(dirs) > 0 {
		diru.SetValue(dirs)
	}
	if len(points) > 0 {
		ptu.SetValue(points)
	}
	if len(spots) > 0 {
		spu.SetValue(spots)
	}
}

/////////////////////////////////////////////////////////////////////////\
//  Standard Light Colors

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

var KiT_LightColors = kit.Enums.AddEnum(LightColorsN, kit.NotBitFlag, nil)

// LightColorMap provides a map of named light colors
var LightColorMap = map[LightColors]gist.Color{
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
