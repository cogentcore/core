// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import "github.com/goki/gi"

// todo: rename gi.Viewport2D -> gi.Viewport

// note: g3n does online variable replacement for number of lights in shader source
// based on actual number of each type of light, for maximum efficiency

// Scene is the overall scenegraph containing nodes as children.
// It renders to its own Viewport, and is functionally analogous to the svg.SVG node.
type Scene struct {
	gi.Viewport2D
	Camera   Camera
	Lights   map[string]Light    `desc:"all lights used in the scene"`
	Mats     map[string]Material `desc:"all materials used in the scene"`
	MatOrder [][]Material        `desc:"materials organized by type and then item order within that"`
	Meshes   map[string]*Mesh    `desc:"all meshes used in the scene"`
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
