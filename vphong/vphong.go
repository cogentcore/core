// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"image"

	"github.com/goki/kigen/ordmap"
	"github.com/goki/vgpu/vgpu"
)

const MaxLights = 8

// Phong implements standard Blinn-Phong rendering pipelines
// in a vgpu System.  The lights must be configured prior
// to starting a new pass.
type Phong struct {
	NLights NLights                 `desc:"number of each type of light"`
	Ambient [MaxLights]AmbientLight `desc:"ambient lights"`
	Dir     [MaxLights]DirLight     `desc:"directional lights"`
	Point   [MaxLights]PointLight   `desc:"point lights"`
	Spot    [MaxLights]SpotLight    `desc:"spot lights"`

	Textures ordmap.Map[string, image.Image] `desc:"texture images"`
	Meshes   ordmap.Map[string, *Mesh]       `desc:"meshes"`

	Sys  vgpu.System   `desc:"rendering system"`
	Surf *vgpu.Surface `desc:"surface if render target"`
}

// ConfigSurface configures the Phong to use given surface as a render target
// maxColors is maximum number of fill colors in palette
func (ph *Phong) ConfigSurface(sf *vgpu.Surface) {
	ph.Surf = sf
	ph.Sys.InitGraphics(sf.GPU, "vdraw.Phong", &sf.Device)
	ph.Sys.ConfigRenderPass(&ph.Surf.Format, vgpu.UndefType)
	sf.SetRenderPass(&ph.Sys.RenderPass)
	ph.ConfigSys()
}

func (ph *Phong) Destroy() {
	ph.Sys.Destroy()
}
