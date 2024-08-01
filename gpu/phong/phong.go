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

// Phong implements standard Blinn-Phong rendering pipelines
// in a gpu System.
// Must Add all Lights, Meshes, Textures and Objects after
// getting a NewPhong, and then call Config() to configure
// everything on the GPU prior to first RenderStart.
//
// If any changes are made to any of these elements after
// initial Config, call the appropriate Config* method
// to update them.
//
// Object data will generally be updated every render frame,
// and it is automatically sync'd up to the GPU during the
// RenderStart call.
//
// Rendering starts with RenderStart, followed by Use* calls
// to specify the render parameters for each item,
// followed by the Render() method that calls the proper
// pipeline's BindDrawVertex based on the render parameters.
type Phong struct {
	// The current camera view and projection matricies.
	// This is used for updating the object WorldMatrix.
	Camera Camera

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
	meshes ordmap.Map[string, *Mesh]

	// Textures holds all of the texture images, managed by AddTexture,
	// DeleteTexture methods.
	textures ordmap.Map[string, *Texture]

	// Objects holds per-object data, keyed by unique name / path id.
	// All objects must be added at start with AddObject,
	// and updated per-pass with UpdateObjects.
	objects ordmap.Map[string, *Object]

	// cameraUpdated is set whenver SetCamera is called.
	// it triggers an up date of the object's WorldMatrix.
	cameraUpdated bool

	// objectUpdated is set whenever SetObject is called,
	// and cleared when the objects have been updated to the GPU.
	objectUpdated bool

	// rendering system
	Sys *gpu.System

	// overall lock on Phong operations, use Lock, Unlock on Phong
	sync.Mutex
}

// NewPhong returns a new Phong system that is ready to be
// configured by adding the relevant elements.
// When done, call Config() to perform initial configuration.
func NewPhong(gp *gpu.GPU, dev *gpu.Device, renderFormat *gpu.TextureFormat) *Phong {
	ph := &Phong{}
	ph.Sys = gpu.NewGraphicsSystem(gp, "phong", dev)
	ph.Sys.ConfigRender(renderFormat, gpu.Depth32)
	// sf.SetRender(&sy.Render)
	ph.configSystem()
	return ph
}

// Release should be called to release all the GPU resources.
func (ph *Phong) Release() {
	ph.Lock()
	defer ph.Unlock()

	if ph.Sys == nil {
		return
	}
	ph.Sys.Release()
	ph.Sys = nil
	ph.meshes.Reset()
	ph.textures.Reset()
	ph.objects.Reset()
}

// Config configures the gpu rendering system after
// everything has been Added for the first time.
// This should generally only be called once,
// and then more specific Config calls made thereafter
// as needed.
func (ph *Phong) Config() *Phong {
	ph.Lock()
	defer ph.Unlock()

	ph.Sys.Config()
	ph.configLights()
	ph.configMeshes()
	ph.configTextures()
	ph.updateObjects()
	return ph
}

// ConfigLights can be called after initial Config
// whenver the Lights data has changed, to sync changes
// up to the GPU.
func (ph *Phong) ConfigLights() *Phong {
	ph.Lock()
	defer ph.Unlock()
	ph.configLights()
	return ph
}

// ConfigMeshes can be called after initial Config
// whenver the Meshes data has changed, to sync changes
// up to the GPU.
func (ph *Phong) ConfigMeshes() *Phong {
	ph.Lock()
	defer ph.Unlock()
	ph.configMeshes()
	return ph
}

// ConfigTextures can be called after initial Config
// whenver the Textures data has changed, to sync changes
// up to the GPU.
func (ph *Phong) ConfigTextures() *Phong {
	ph.Lock()
	defer ph.Unlock()
	ph.configTextures()
	return ph
}

///////////////////////////////////////////////////
// Rendering

// RenderStart starts the render pass, returning the
// CommandEncoder and RenderPassEncoder used for encoding
// the rendering commands for this pass.
// Pass the TextureView to render into (e.g., from Surface).
// This also ensures that all updated object data from SetObject*
// calls is transferred to the GPU.
func (ph *Phong) RenderStart(view *wgpu.TextureView) (*wgpu.CommandEncoder, *wgpu.RenderPassEncoder) {
	ph.Lock()
	defer ph.Unlock()

	ph.updateObjects()

	cmd := ph.Sys.NewCommandEncoder()
	rp := ph.Sys.BeginRenderPass(cmd, view)
	return cmd, rp
}

// Render does one step of rendering given current Use* settings,
// which can be updated in between subsequent Render calls.
func (ph *Phong) Render(rp *wgpu.RenderPassEncoder) {
	ph.Lock()
	defer ph.Unlock()

	ph.RenderOneColor(rp)
	return

	switch {
	case ph.UseTexture:
		ph.RenderTexture(rp)
	case ph.UseVertexColor:
		ph.RenderVertexColor(rp)
	default:
		ph.RenderOneColor(rp)
	}
}

// RenderOneColor renders current settings to onecolor pipeline.
func (ph *Phong) RenderOneColor(rp *wgpu.RenderPassEncoder) {
	pl := ph.Sys.GraphicsPipelines["onecolor"]
	pl.BindPipeline(rp)
	pl.BindDrawVertex(rp)
}

// RenderTexture renders current settings to texture pipeline
func (ph *Phong) RenderTexture(rp *wgpu.RenderPassEncoder) {
	pl := ph.Sys.GraphicsPipelines["texture"]
	pl.BindPipeline(rp)
	pl.BindDrawVertex(rp)
}

// RenderVertexColor renders current settings to vertexcolor pipeline
func (ph *Phong) RenderVertexColor(rp *wgpu.RenderPassEncoder) {
	pl := ph.Sys.GraphicsPipelines["pervertex"]
	pl.BindPipeline(rp)
	pl.BindDrawVertex(rp)
}
