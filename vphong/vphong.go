// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"sync"

	"github.com/goki/kigen/ordmap"
	"github.com/goki/vgpu/vgpu"
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
	NLights   NLights                 `desc:"number of each type of light"`
	Ambient   [MaxLights]AmbientLight `desc:"ambient lights"`
	Dir       [MaxLights]DirLight     `desc:"directional lights"`
	Point     [MaxLights]PointLight   `desc:"point lights"`
	Spot      [MaxLights]SpotLight    `desc:"spot lights"`
	Wireframe bool                    `def:"false" desc:"render using wireframe instead of filled polygons -- this must be set prior to configuring the Phong rendering system"`

	Cur      CurRender                    `desc:"state for current rendering"`
	Meshes   ordmap.Map[string, *Mesh]    `desc:"meshes -- holds all the mesh data -- must be configured prior to rendering"`
	Textures ordmap.Map[string, *Texture] `desc:"textures -- must be configured prior to rendering -- a maximum of 16 textures is supported for full cross-platform portability"`
	Colors   ordmap.Map[string, *Colors]  `desc:"colors, optionally available for looking up by name -- not used directly in rendering"`

	Sys    vgpu.System `desc:"rendering system"`
	UpdtMu sync.Mutex  `view:"-" copy:"-" json:"-" xml:"-" desc:"mutex on updating"`
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
