// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

//go:generate core generate

import (
	"sync"

	"cogentcore.org/core/base/ordmap"
	"cogentcore.org/core/gpu"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// MaxLights is upper limit on number of any given type of light
const MaxLights = 8

// Phong implements standard Blinn-Phong rendering pipelines in a vgpu System.
// Must Add all Lights, Meshes, Colors, Textures first, and call
// Config() to configure everything prior to first RenderStart.
//
// Meshes are configured initially with numbers of points, then
// after Config(), points are set by calling MeshFloatsBy* and
// assigning values.
//
// If any changes are made to numbers or sizes of anything,
// you must call Config() again.
//
// Changes to data only can be synced by calling Sync()
//
// Rendering starts with RenderStart, followed by Use* calls
// to specify the parameters for each item, and then a Draw call
// to add the rendering command, followed by RenderEnd.
type Phong struct {

	// number of each type of light
	NLights NLights

	// ambient lights
	Ambient [MaxLights]AmbientLight

	// directional lights
	Dir [MaxLights]DirLight

	// point lights
	Point [MaxLights]PointLight

	// spot lights
	Spot [MaxLights]SpotLight

	// a texture was selected for next draw, if true, overrides other options
	UseTexture bool

	// a per-vertex color was selected for next draw
	UseVertexColor bool

	// render using wireframe instead of filled polygons.
	// this must be set prior to configuring the Phong rendering system.
	Wireframe bool `default:"false"`

	// Meshes holds all of the mesh data, managed by AddMesh, DeleteMesh
	// methods.
	Meshes ordmap.Map[string, *Mesh]

	// Textures holds all of the texture images, managed by AddTexture,
	// DeleteTexture methods.
	Textures ordmap.Map[string, *Texture]

	// Objects holds per-object data, keyed by unique name / path id.
	// All objects must be added at start with AddObject,
	// and updated per-pass with UpdateObject.
	Objects ordmap.Map[string, *Object]

	// rendering system
	Sys *gpu.System

	// overall lock on Phong operations, use Lock, Unlock on Phong
	sync.Mutex
}

func (ph *Phong) Release() {
	ph.Sys.Release()
}

// Config configures the gpu rendering system after
// everything has been Added for the first time.
func (ph *Phong) Config() {
	ph.ConfigMeshesTextures()
	ph.Lock()
	ph.Sys.Config()
	ph.ConfigLights()
	ph.Unlock()
}

// ConfigMeshesTextures configures the Meshes and Textures based
// on everything added in the Phong config, prior to Sys.Config()
// which does host allocation.
func (ph *Phong) ConfigMeshesTextures() {
	ph.Lock()
	ph.ConfigMeshes()
	ph.ConfigTextures()
	ph.Unlock()
}

// Sync synchronizes any changes in val data up to GPU device memory.
// any changes in numbers or sizes of any element requires a Config call.
func (ph *Phong) Sync() {
	ph.Lock()
	// todo!
	ph.Unlock()
}

///////////////////////////////////////////////////
// Rendering

// Render does one step of rendering given current Use* settings
func (ph *Phong) Render(rp *wgpu.RenderPassEncoder) {
	ph.Lock()
	defer ph.Unlock()

	switch {
	case ph.UseTexture:
		ph.RenderTexture(rp)
	case ph.UseVertexColor:
		ph.RenderVertexColor(rp)
	default:
		ph.RenderOneColor(rp)
	}
}

// RenderTexture renders current settings to texture pipeline
func (ph *Phong) RenderTexture(rp *wgpu.RenderPassEncoder) {
	pl := ph.Sys.GraphicsPipelines["texture"]
	pl.BindDrawVertex(rp)
}

// RenderOneColor renders current settings to onecolor pipeline.
func (ph *Phong) RenderOneColor(rp *wgpu.RenderPassEncoder) {
	pl := ph.Sys.GraphicsPipelines["onecolor"]
	pl.BindDrawVertex(rp)
}

// RenderVertexColor renders current settings to vertexcolor pipeline
func (ph *Phong) RenderVertexColor(rp *wgpu.RenderPassEncoder) {
	pl := ph.Sys.GraphicsPipelines["pervertex"]
	pl.BindDrawVertex(rp)
}