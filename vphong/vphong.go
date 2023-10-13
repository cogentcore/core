// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"sync"

	"goki.dev/ordmap"
	"goki.dev/vgpu/v2/vgpu"
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

	// render using wireframe instead of filled polygons -- this must be set prior to configuring the Phong rendering system
	Wireframe bool `def:"false"`

	// state for current rendering
	Cur CurRender

	// meshes -- holds all the mesh data -- must be configured prior to rendering
	Meshes ordmap.Map[string, *Mesh]

	// textures -- must be configured prior to rendering -- a maximum of 16 textures is supported for full cross-platform portability
	Textures ordmap.Map[string, *Texture]

	// colors, optionally available for looking up by name -- not used directly in rendering
	Colors ordmap.Map[string, *Colors]

	// rendering system
	Sys vgpu.System

	// mutex on updating
	UpdtMu sync.Mutex `view:"-" copy:"-" json:"-" xml:"-"`
}

func (ph *Phong) Destroy() {
	ph.Sys.Destroy()
}

// Config configures everything after everything has been Added
func (ph *Phong) Config() {
	ph.ConfigMeshesTextures()
	ph.UpdtMu.Lock()
	ph.Sys.Config()

	ph.ConfigLights()
	ph.AllocTextures()
	ph.Sys.Mem.SyncToGPU()
	ph.UpdtMu.Unlock()
}

// ConfigMeshesTextures configures the Meshes and Textures based
// on everything added in the Phong config, prior to Sys.Config()
// which does host allocation.
func (ph *Phong) ConfigMeshesTextures() {
	ph.UpdtMu.Lock()
	ph.Sys.Mem.Free()
	ph.ConfigMeshes()
	ph.ConfigTextures()
	ph.UpdtMu.Unlock()
}

// Sync synchronizes any changes in val data up to GPU device memory.
// any changes in numbers or sizes of any element requires a Config call.
func (ph *Phong) Sync() {
	ph.UpdtMu.Lock()
	ph.Sys.Mem.SyncToGPU()
	ph.UpdtMu.Unlock()
}

///////////////////////////////////////////////////
// Rendering

// Render does one step of rendering given current Use* settings
func (ph *Phong) Render() {
	ph.UpdtMu.Lock()
	sy := &ph.Sys
	cmd := sy.CmdPool.Buff
	sy.CmdBindVars(cmd, 0) // updates all dynamics
	switch {
	case ph.Cur.UseTexture:
		ph.RenderTexture()
	case ph.Cur.UseVtxColor:
		ph.RenderVtxColor()
	default:
		ph.RenderOnecolor()
	}
	ph.UpdtMu.Unlock()
}
