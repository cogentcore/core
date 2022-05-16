// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"fmt"
	"image/color"
	"log"
	"unsafe"

	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
)

// Color describes the surface colors for Phong lighting model
type Color struct {
	Color       mat32.Vec4 `desc:"main reflective color: reflected from lights"`
	Emissive    mat32.Vec3 `desc:"color that surface emits"`
	pad1        float32
	Specular    mat32.Vec3 `desc:"shiny reflection color"`
	pad2        float32
	ShinyBright mat32.Vec3 `desc:"x = Shiny, y = Bright"`
	pad3        float32
}

// NewGoColor3 returns a mat32.Vec3 from Go standard color.Color
func NewGoColor3(clr color.Color) mat32.Vec3 {
	v3 := mat32.Vec3{}
	SetGoColor3(&v3, clr)
	return v3
}

// SetGoColor3 sets a mat32.Vec3 from Go standard color.Color
func SetGoColor3(v3 *mat32.Vec3, clr color.Color) {
	r, g, b, _ := clr.RGBA()
	v3.X = float32(r) / 0xffff
	v3.Y = float32(g) / 0xffff
	v3.Z = float32(b) / 0xffff
}

// NewGoColor4 returns a mat32.Vec4 from Go standard color.Color
func NewGoColor4(clr color.Color) mat32.Vec4 {
	v4 := mat32.Vec4{}
	SetGoColor4(&v4, clr)
	return v4
}

// SetGoColor4 sets a mat32.Vec4 from Go standard color.Color
func SetGoColor4(v4 *mat32.Vec4, clr color.Color) {
	r, g, b, a := clr.RGBA()
	v4.X = float32(r) / 0xffff
	v4.Y = float32(g) / 0xffff
	v4.Z = float32(b) / 0xffff
	v4.W = float32(a) / 0xffff
}

// NewGoColor sets the colors from standard Go colors
func NewColors(clr, emis, spec color.Color, shiny, bright float32) *Color {
	cl := &Color{}
	cl.SetColors(clr, emis, spec, shiny, bright)
	return cl
}

// SetColors sets the colors from standard Go colors
func (cl *Color) SetColors(clr, emis, spec color.Color, shiny, bright float32) {
	SetGoColor4(&cl.Color, clr)
	SetGoColor3(&cl.Emissive, emis)
	SetGoColor3(&cl.Specular, spec)
	cl.ShinyBright.X = shiny
	cl.ShinyBright.Y = bright
}

// AddColor adds to list of colors
func (ph *Phong) AddColor(name string, clr *Color) {
	ph.Colors.Add(name, clr)
}

// AllocColors allocates vals for colors
func (ph *Phong) AllocColors() {
	vars := ph.Sys.Vars()
	clrset := vars.SetMap[int(ColorSet)]
	clrset.ConfigVals(ph.Colors.Len())
}

// ConfigColors configures the rendering for the colors that have been added.
func (ph *Phong) ConfigColors() {
	vars := ph.Sys.Vars()
	clrset := vars.SetMap[int(ColorSet)]
	for i, kv := range ph.Colors.Order {
		_, clr, _ := clrset.ValByIdxTry("Color", i)
		clr.CopyBytes(unsafe.Pointer(kv.Val))
	}
}

// UseColorIdx selects color by index for current render step
func (ph *Phong) UseColorIdx(idx int) error {
	vars := ph.Sys.Vars()
	vars.BindDynValIdx(int(ColorSet), "Color", idx)
	ph.Cur.ColorIdx = idx // todo: range check
	return nil
}

// UseColorName selects color by name for current render step
func (ph *Phong) UseColorName(name string) error {
	idx, ok := ph.Colors.IdxByKey(name)
	if !ok {
		err := fmt.Errorf("vphong:UseColorName -- name not found: %s", name)
		if vgpu.TheGPU.Debug {
			log.Println(err)
		}
	}
	return ph.UseColorIdx(idx)
}

// RenderOnecolor renders current settings to onecolor pipeline
func (ph *Phong) RenderOnecolor() {
	sy := &ph.Sys
	cmd := sy.CmdPool.Buff
	pl := sy.PipelineMap["onecolor"]
	pl.BindDrawVertex(cmd, ph.Cur.DescIdx)
}
