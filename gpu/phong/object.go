// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"fmt"
	"image/color"
	"log"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
)

// Object is the object-specific data: Colors and transform Matrix.
// It must be updated on a per-object basis prior to each render pass.
// It is 8 vec4 in size = 8 * 4 * 4 = 128 bytes.
type Object struct {
	Colors

	// Matrix specifies the transformation matrix for this specific Object.
	Matrix math32.Matrix4
}

// NewObject returns a new object with given matrix and colors.
// Texture defaults to using full texture with 0 offset.
func NewObject(mtx *math32.Matrix4, clr, emis color.Color, shiny, reflect, bright float32) *Object {
	ob := &Object{}
	ob.Matrix = *mtx
	ob.SetColors(clr, emis, shiny, reflect, bright)
	ob.TextureRepeatOff.Set(1, 1, 0, 0)
	return ob
}

// AddObject adds a Object with given unique name identifier.
func (ph *Phong) AddObject(name string, ob *Object) {
	ph.Objects.Add(name, ob)
}

// DeleteObject deletes Object with name
func (ph *Phong) DeleteObject(name string) {
	ph.Objects.DeleteKey(name)
}

// UseObjectName selects object by name for current render step
// If object has per-vertex colors, these are selected for rendering,
// and texture is turned off.  UseTexture* after this to override.
func (ph *Phong) UseObjectName(name string) error {
	idx, ok := ph.Objects.IndexByKeyTry(name)
	if !ok {
		err := fmt.Errorf("phong:UseObjectName -- name not found: %s", name)
		if gpu.Debug {
			log.Println(err)
		}
	}
	return ph.UseObjectIndex(idx)
}

// UseObjectIndex selects object by index for current render step.
// If object has per-vertex colors, these are selected for rendering,
// and texture is turned off.  UseTexture* after this to override.
func (ph *Phong) UseObjectIndex(idx int) error {
	ph.Sys.Vars.SetDynamicIndex(int(ObjectsGroup), "Object", idx)
	return nil
}
