// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"github.com/goki/kigen/ordmap"
	"github.com/goki/vgpu/vgpu"
)

// MaxLights is upper limit on number of any given type of light
const MaxLights = 8

// CurRender holds info about the current render as updated by
// Use* methods -- determines which pipeline is used.
// Default is single color.
type CurRender struct {
	DescIdx     int     `desc:"descriptor index"`
	UseTexture  bool    `desc:"a texture was selected -- if true, overrides other options"`
	UseVtxColor bool    `desc:"a per-vertex color was selected"`
	MtxsIdx     int     `desc:"index of currently selected matrix (dynamically bound)"`
	ColorIdx    int     `desc:"index of currently-selected color (dynamically bound)"`
	TexIdx      int     `desc:"index of currently-selected texture"`
	TexPush     TexPush `desc:"texture push constant data"`
}

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
//
type Phong struct {
	NLights NLights                 `desc:"number of each type of light"`
	Ambient [MaxLights]AmbientLight `desc:"ambient lights"`
	Dir     [MaxLights]DirLight     `desc:"directional lights"`
	Point   [MaxLights]PointLight   `desc:"point lights"`
	Spot    [MaxLights]SpotLight    `desc:"spot lights"`

	Cur      CurRender                    `desc:"state for current rendering"`
	Mtxs     ordmap.Map[string, *Mtxs]    `desc:"model-view-projection matrixes per object"`
	Meshes   ordmap.Map[string, *Mesh]    `desc:"meshes"`
	Colors   ordmap.Map[string, *Color]   `desc:"colors"`
	Textures ordmap.Map[string, *Texture] `desc:"textures"`

	Sys  vgpu.System   `desc:"rendering system"`
	Surf *vgpu.Surface `desc:"surface if render target"`
}

// ConfigSurface configures the Phong to use given surface as a render target
// maxColors is maximum number of fill colors in palette
func (ph *Phong) ConfigSurface(sf *vgpu.Surface) {
	ph.Surf = sf
	ph.Sys.InitGraphics(sf.GPU, "vphong.Phong", &sf.Device)
	ph.Sys.ConfigRenderPass(&ph.Surf.Format, vgpu.UndefType)
	sf.SetRenderPass(&ph.Sys.RenderPass)
	ph.ConfigSys()
}

func (ph *Phong) Destroy() {
	ph.Sys.Destroy()
}

// Config configures everything after everything has been Added
func (ph *Phong) Config() {
	ph.Alloc() // allocate all vals
	ph.Sys.Config()

	ph.ConfigMeshes()
	ph.ConfigLights()
	ph.ConfigMtxs()
	ph.ConfigColors()
	ph.ConfigTextures()
	ph.Sys.Mem.SyncToGPU()
}

// Alloc allocate all vals based on currently-added
// Mesh, Color, Texture
func (ph *Phong) Alloc() {
	ph.AllocMeshes()
	ph.AllocMtxs()
	ph.AllocColors()
	ph.AllocTextures()
}

// Sync synchronizes any changes in val data up to GPU device memory.
// any changes in numbers or sizes of any element requires a Config call.
func (ph *Phong) Sync() {
	ph.Sys.Mem.SyncToGPU()
}

///////////////////////////////////////////////////
// Rendering

// Render does one step of rendering given current Use* settings
func (ph *Phong) Render() {
	sy := &ph.Sys
	cmd := sy.CmdPool.Buff
	sy.CmdBindVars(cmd, 0) // updates all dynamics
	switch {
	case ph.Cur.UseTexture:
		ph.RenderTexture()
	}
}
