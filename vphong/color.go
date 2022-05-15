// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"fmt"
	"log"
	"unsafe"

	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
)

// Color describes the surface colors for Phong lighting model
type Color struct {
	Color       mat32.Vec3 `desc:"main reflective color: reflected from lights"`
	pad0        float32
	Emissive    mat32.Vec3 `desc:"color that surface emits"`
	pad1        float32
	Specular    mat32.Vec3 `desc:"shiny reflection color"`
	pad2        float32
	ShinyBright mat32.Vec3 `desc:"x = Shiny, y = Bright"`
	pad3        float32
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
		clr.CopyBytes(unsafe.Pointer(&kv.Val))
	}
}

// UseColorIdx selects color by index for current render step
func (ph *Phong) UseColorIdx(idx int) error {
	tex := ph.Colors.ValByIdx(idx)
	vars := ph.Sys.Vars()
	vars.BindDynValByIdx(int(ColorSet), "Color", idx)
	ph.Cur.ColorIdx = idx // todo: range check
	ph.Cur.UseColor = true
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
