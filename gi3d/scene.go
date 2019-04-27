// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi"
	"github.com/goki/ki/ki"
)

// Scene is the overall scenegraph containing nodes as children.
// It renders to its own Viewport, and is functionally analogous to the svg.SVG node.
type Scene struct {
	gi.Viewport2D
	Camera   Camera
	Lights   map[string]Light    `desc:"all lights used in the scene"`
	Mats     map[string]Material `desc:"all materials used in the scene"`
	MatOrder [][]Material        `desc:"materials organized by type and then item order within that"`
	Meshes   map[string]*Mesh    `desc:"all meshes used in the scene"`
	Rends    Renderers           `desc:"rendering programs"`
}

// DeleteUnusedMats deletes all unused materials
func (sc *Scene) DeleteUnusedMats() {
	// used := make(map[string]struct{})
	// iterate over scene, add to used, then iterate over mats and if not used, delete.
}

// DeleteUnusedMeshes deletes all unused meshes
func (sc *Scene) DeleteUnusedMats() {
	// used := make(map[string]struct{})
	// iterate over scene, add to used, then iterate over mats and if not used, delete.
}

// Render renders the scene
func (sc *Scene) Render() {
	sc.UpdateMVPMatrix()
}

// UpdateMVPMatrix does a full update of MVP matrix for all visible scene elements
func (sc *Scene) UpdateMVPMatrix() {
	sc.Camera.UpdateMatrix()
	nb.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sc.This() {
			return true
		}
		nii, ni := KiToNode3D(k)
		if nii == nil {
			return false // going into a different type of thing, bail
		}
		if ni.IsInvisible() {
			return false
		}
		nii.UpdateMVPMatrix(&sc.Camera.ViewMatrix, &sc.Camera.PrjnMatrix)
		return true
	})
}
