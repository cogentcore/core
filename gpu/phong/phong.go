// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

//go:generate core generate

import (
	"image"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/ordmap"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/shape"
	"github.com/cogentcore/webgpu/wgpu"
)

// MaxLights is upper limit on number of any given type of light
const MaxLights = 8

// Phong implements standard Blinn-Phong rendering pipelines
// in a gpu GraphicsSystem.
// Add Lights and call SetMesh, SetTexture, SetObject to config.
//
// Rendering starts with RenderStart, followed by Use* calls
// to specify the render parameters for each item,
// followed by the Render() method that calls the proper
// pipeline's render methods.
type Phong struct {
	// The current camera view and projection matricies.
	// This is used for updating the object WorldMatrix.
	Camera Camera

	// number of each type of light
	NLights NLights

	// ambient lights
	Ambient [MaxLights]Ambient

	// directional lights
	Directional [MaxLights]Directional

	// point lights
	Point [MaxLights]Point

	// spot lights
	Spot [MaxLights]Spot

	// a texture was selected for next draw via [UseTexture].
	// if true, overrides other options.
	UseCurTexture bool

	// a per-vertex color was selected for next draw.
	UseVertexColor bool

	// render using wireframe instead of filled polygons.
	// this must be set prior to configuring the Phong rendering system.
	// note: not currently supported in WebGPU.
	Wireframe bool `default:"false"`

	// Meshes holds all of the mesh data, managed by [SetMesh],
	// [ResetMeshes] methods.
	meshes ordmap.Map[string, *shape.MeshData]

	// Textures holds all of the texture images, managed by [SetTexture],
	// [ResetTextures] methods.
	textures ordmap.Map[string, *Texture]

	// Objects holds per-object data, keyed by unique name / path id.
	// All objects must be added in a pre-Render pass via [SetObject].
	objects ordmap.Map[string, *Object]

	// cameraUpdated is set whenver SetCamera is called.
	// it triggers an up date of the object's WorldMatrix.
	cameraUpdated bool

	// objectUpdated is set whenever SetObject is called,
	// and cleared when the objects have been updated to the GPU.
	objectUpdated bool

	// lightsUpdated indicates a change has been made to lights.
	lightsUpdated bool

	// rendering system
	System *gpu.GraphicsSystem

	// overall lock on Phong operations, use Lock, Unlock on Phong
	sync.Mutex
}

// NewPhong returns a new Phong system that is ready to be
// configured by calls to SetMesh, SetTexture, SetObject,
// in addition to adding lights.
// Renderer can either be a Surface or a RenderTexture.
func NewPhong(gp *gpu.GPU, rd gpu.Renderer) *Phong {
	ph := &Phong{}
	ph.System = gpu.NewGraphicsSystem(gp, "phong", rd)
	ph.configGraphicsSystem()
	return ph
}

// Release should be called to release all the GPU resources.
func (ph *Phong) Release() {
	ph.Lock()
	defer ph.Unlock()

	if ph.System == nil {
		return
	}
	ph.System.Release()
	ph.System = nil
	ph.meshes.Reset()
	ph.textures.Reset()
	ph.objects.Reset()
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

// ResetAll resets all the dynamic resources: Objects, Meshes, Textures
// and Lights.
func (ph *Phong) ResetAll() {
	ph.ResetObjects()
	ph.ResetMeshes()
	ph.ResetTextures()
	ph.ResetLights()
}

///////////////////////////////////////////////////
// Rendering

// RenderStart starts the render pass, returning the
// RenderPassEncoder used for encoding the rendering commands
// for this pass.
// This also ensures that all updated object data from SetObject*
// calls is transferred to the GPU.
func (ph *Phong) RenderStart() (*wgpu.RenderPassEncoder, error) {
	ph.Lock()
	defer ph.Unlock()

	ph.configLights()       // if needed
	ph.configDummyTexture() // if needed
	ph.updateObjects()

	if rt, ok := ph.System.Renderer.(*gpu.RenderTexture); ok {
		rt.CurrentFrame().ConfigReadBuffer()
	}

	return ph.System.BeginRenderPass()
}

// RenderEnd ends the render pass. Must be paired after RenderStart.
func (ph *Phong) RenderEnd(rp *wgpu.RenderPassEncoder) {
	rp.End()
	ph.System.EndRenderPass(rp)
}

// RenderEndGrabImage grabs the rendered image from rendering.
// The command to read the image must be inserted at the end of the
// render commands, so this is an alternative to standard RenderEnd.
// This only works if the system Renderer is a [RenderTexture],
// otherwise it will return nil.
func (ph *Phong) RenderEndGrabImage(rp *wgpu.RenderPassEncoder) *image.NRGBA {
	rp.End()
	sy := ph.System
	rt, ok := sy.Renderer.(*gpu.RenderTexture)
	if !ok {
		sy.EndRenderPass(rp)
		return nil
	}
	rt.ReadFrame(sy.CommandEncoder)
	sy.EndRenderPass(rp)
	img := errors.Log1(rt.CurrentFrame().ReadGoImage())
	return img
}

// Render does one step of rendering given current Use* settings,
// which can be updated in between subsequent Render calls.
func (ph *Phong) Render(rp *wgpu.RenderPassEncoder) {
	ph.Lock()
	defer ph.Unlock()

	switch {
	case ph.UseCurTexture:
		ph.RenderTexture(rp)
	case ph.UseVertexColor:
		ph.RenderVertexColor(rp)
	default:
		ph.RenderOneColor(rp)
	}
}

// RenderOneColor renders current settings to onecolor pipeline.
func (ph *Phong) RenderOneColor(rp *wgpu.RenderPassEncoder) {
	pl := ph.System.GraphicsPipelines["onecolor"]
	pl.BindPipeline(rp)
	pl.BindDrawIndexed(rp)
}

// RenderTexture renders current settings to texture pipeline
func (ph *Phong) RenderTexture(rp *wgpu.RenderPassEncoder) {
	pl := ph.System.GraphicsPipelines["texture"]
	pl.BindPipeline(rp)
	pl.BindDrawIndexed(rp)
}

// RenderVertexColor renders current settings to vertexcolor pipeline
func (ph *Phong) RenderVertexColor(rp *wgpu.RenderPassEncoder) {
	pl := ph.System.GraphicsPipelines["pervertex"]
	pl.BindPipeline(rp)
	pl.BindDrawIndexed(rp)
}
