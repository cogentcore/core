// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"fmt"
	"image/color"
	"log"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
)

// Object is the object-specific data: Colors and transform Matrix.
// It must be updated on a per-object basis prior to each render pass.
// It is 8 vec4 in size = 8 * 4 * 4 = 128 bytes.
type Object struct {
	Colors

	// Matrix specifies the transformation matrix for this specific Object ("model").
	Matrix math32.Matrix4

	// WorldMatrix is the transpose of the inverse of the
	// Camera.View matrix * Object "model" Matrix, used to
	// compute the proper normals. WebGPU does not
	// have the transpose function.
	WorldMatrix math32.Matrix4
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
	ph.Lock()
	defer ph.Unlock()

	ph.objects.Add(name, ob)
}

// DeleteObject deletes Object with name
func (ph *Phong) DeleteObject(name string) {
	ph.Lock()
	defer ph.Unlock()

	ph.objects.DeleteKey(name)
}

// ResetObjects resets the objects for reconfiguring
func (ph *Phong) ResetObjects() {
	ph.Lock()
	defer ph.Unlock()

	ph.objects.Reset()
	// updateObjects is auto-resetting
}

func (ph *Phong) object(name string) *Object {
	ob, ok := ph.objects.ValueByKeyTry(name)
	if !ok {
		err := fmt.Errorf("phong:Object: name not found: %s", name)
		errors.Log(err)
	}
	return ob
}

func (ph *Phong) setWorldMatrix(ob *Object) {
	mvm := math32.Matrix3FromMatrix4(ph.Camera.View.Mul(&ob.Matrix))
	nm := mvm.Inverse().Transpose()
	ob.WorldMatrix.SetFromMatrix3(&nm)
}

// SetObject sets the updated object data for given object name.
// This must be called for any object updates _prior_ to the next
// render pass.  All of the object data must be transferred to the
// GPU if any are updated, so in general it is fine to update
// everything every time, just in case anything changed.
func (ph *Phong) SetObject(name string, mtx *math32.Matrix4, clr, emis color.Color, shiny, reflect, bright float32) *Object {
	ob := ph.object(name)
	if ob == nil {
		return nil
	}
	ob.Matrix = *mtx
	ob.SetColors(clr, emis, shiny, reflect, bright)
	ph.objectUpdated = true
	return ob
}

// SetObjectMatrix sets the updated object matrix for given object name.
// This must be called for any object updates _prior_ to the next
// render pass.  All of the object data must be transferred to the
// GPU if any are updated, so in general it is fine to update
// everything every time, just in case anything changed.
func (ph *Phong) SetObjectMatrix(name string, mtx *math32.Matrix4) *Object {
	ob := ph.object(name)
	if ob == nil {
		return nil
	}
	ob.Matrix = *mtx
	ph.objectUpdated = true
	return ob
}

// SetObjectColor sets the updated object colors for given object name.
// This must be called for any object updates _prior_ to the next
// render pass.  All of the object data must be transferred to the
// GPU if any are updated, so in general it is fine to update
// everything every time, just in case anything changed.
func (ph *Phong) SetObjectColor(name string, clr, emis color.Color, shiny, reflect, bright float32) *Object {
	ob := ph.object(name)
	if ob == nil {
		return nil
	}
	ob.SetColors(clr, emis, shiny, reflect, bright)
	ph.objectUpdated = true
	return ob
}

// UseObjectName selects object by name for current render step
// If object has per-vertex colors, these are selected for rendering,
// and texture is turned off.  UseTexture* after this to override.
func (ph *Phong) UseObjectName(name string) error {
	idx, ok := ph.objects.IndexByKeyTry(name)
	if !ok {
		err := fmt.Errorf("phong:UseObjectName: name not found: %s", name)
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
	ph.Lock()
	defer ph.Unlock()

	ph.System.Vars().SetDynamicIndex(int(ObjectGroup), "Object", idx)
	return nil
}

// updateObjects updates the object specific data to the GPU,
// updating the WorldMatrix based on current Camera settings first.
// This is called in the RenderStart function.
func (ph *Phong) updateObjects() {
	if !(ph.objectUpdated || ph.cameraUpdated) {
		return
	}
	vl := ph.System.Vars().ValueByIndex(int(ObjectGroup), "Object", 0)
	vl.SetDynamicN(ph.objects.Len())
	for i, kv := range ph.objects.Order {
		ob := kv.Value
		ph.setWorldMatrix(ob)
		gpu.SetDynamicValueFrom(vl, i, []Object{*ob})
	}
	vl.WriteDynamicBuffer()
	ph.objectUpdated = false
	ph.cameraUpdated = false
}
