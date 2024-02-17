// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"fmt"
	"image/color"
	"log"

	"cogentcore.org/core/mat32"
)

// Colors are the material colors with padding for direct uploading to shader
type Colors struct {

	// main color of surface, used for both ambient and diffuse color in standard Phong model -- alpha component determines transparency -- note that transparent objects require more complex rendering
	Color mat32.Vec4

	// X = shininess spread factor, Y = shine reflection factor, Z = brightness factor:  shiny = specular shininess factor -- how focally the surface shines back directional light -- this is an exponential factor, with 0 = very broad diffuse reflection, and higher values (typically max of 128) having a smaller more focal specular reflection.  Shine reflect = 1 for full shine white reflection (specular) color, 0 = no shine reflection.  bright = overall multiplier on final computed color value -- can be used to tune the overall brightness of various surfaces relative to each other for a given set of lighting parameters.  W is used for Tex idx.
	ShinyBright mat32.Vec4

	// color that surface emits independent of any lighting -- i.e., glow -- can be used for marking lights with an object
	Emissive mat32.Vec4
}

// NewGoColor sets the colors from standard Go colors
func NewColors(clr, emis color.Color, shiny, reflect, bright float32) *Colors {
	cl := &Colors{}
	cl.SetColors(clr, emis, shiny, reflect, bright)
	return cl
}

// SetColors sets the colors from standard Go colors
func (cl *Colors) SetColors(clr, emis color.Color, shiny, reflect, bright float32) {
	cl.Color = mat32.NewVec4Color(clr).SRGBToLinear()
	cl.Emissive = mat32.NewVec4Color(emis).SRGBToLinear()
	cl.ShinyBright.X = shiny
	cl.ShinyBright.Y = reflect
	cl.ShinyBright.Z = bright
}

// AddColor adds to list of colors, which can be use for a materials library
func (ph *Phong) AddColor(name string, clr *Colors) {
	ph.Colors.Add(name, clr)
}

// UseColorIdx selects color by index for current render step
func (ph *Phong) UseColorIdx(index int) error {
	if err := ph.Colors.IndexIsValid(index); err != nil {
		log.Println(err)
		return err
	}
	clr := ph.Colors.ValueByIndex(index)
	ph.Cur.Color = *clr
	return nil
}

// UseColorName selects color by name for current render step
func (ph *Phong) UseColorName(name string) error {
	idx, ok := ph.Colors.IndexByKeyTry(name)
	if !ok {
		err := fmt.Errorf("vphong:UseColorName -- name not found: %s", name)
		log.Println(err)
		return err
	}
	clr := ph.Colors.ValueByIndex(idx)
	ph.Cur.Color = *clr
	return nil
}

// UseColors sets the color values for current render step
func (ph *Phong) UseColor(clr, emis color.Color, shiny, reflect, bright float32) {
	ph.Cur.Color.SetColors(clr, emis, shiny, reflect, bright)
}

// RenderOnecolor renders current settings to onecolor pipeline
func (ph *Phong) RenderOnecolor() {
	sy := &ph.Sys
	cmd := sy.CmdPool.Buff
	pl := sy.PipelineMap["onecolor"]
	push := ph.Cur.NewPush()
	ph.Push(pl, push)
	pl.BindDrawVertex(cmd, ph.Cur.DescIdx)
}

// RenderVtxColor renders current settings to vertexcolor pipeline
func (ph *Phong) RenderVtxColor() {
	sy := &ph.Sys
	cmd := sy.CmdPool.Buff
	pl := sy.PipelineMap["pervertex"]
	push := ph.Cur.NewPush()
	ph.Push(pl, push)
	pl.BindDrawVertex(cmd, ph.Cur.DescIdx)
}
