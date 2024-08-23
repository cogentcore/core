// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"fmt"

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
	// have the transpose function.  This is managed entirely by the
	// Phong system and is not set by the user.
	worldMatrix math32.Matrix4
}

// NewObject returns a new object with given matrix and colors.
func NewObject(mtx *math32.Matrix4, clr *Colors) *Object {
	ob := &Object{}
	ob.Matrix = *mtx
	ob.Colors = *clr
	return ob
}

// SetObject sets Object data with given unique name identifier.
// If object already exists, then data is updated if different.
func (ph *Phong) SetObject(name string, ob *Object) {
	ph.Lock()
	defer ph.Unlock()

	idx, ok := ph.objects.Map[name]
	if !ok {
		ph.objects.Add(name, ob)
		ph.objectUpdated = true
		// note: allocation of DynamicN happens in updateObjects pass.
		return
	}
	cob := ph.objects.Order[idx].Value
	if *ob != *cob {
		ph.objects.Order[idx].Value = ob
		ph.objectUpdated = true
	}
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
	ob.worldMatrix.SetFromMatrix3(&nm)
}

// UseObject selects object by name for current render step.
// Object must have already been added / updated via [SetObject].
// If object has per-vertex colors, these are selected for rendering,
// and texture is turned off.  UseTexture* after this to override.
func (ph *Phong) UseObject(name string) error {
	ph.Lock()
	defer ph.Unlock()

	idx, ok := ph.objects.IndexByKeyTry(name)
	if !ok {
		return errors.Log(fmt.Errorf("phong:UseObject: name not found: %s", name))
	}
	ph.System.Vars().SetDynamicIndex(int(ObjectGroup), "Object", idx)
	return nil
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
	vl, _ := ph.System.Vars().ValueByIndex(int(ObjectGroup), "Object", 0)
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
