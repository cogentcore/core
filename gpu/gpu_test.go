// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"
	"testing"

	_ "cogentcore.org/core/base/iox/imagex"
	"github.com/stretchr/testify/assert"
)

func TestGPUTriangle(t *testing.T) {
	gp, dev, err := NoDisplayGPU()
	assert.NoError(t, err)
	sz := image.Point{480, 320}
	rt := NewRenderTexture(gp, dev, sz, 4, Depth32)
	sy := NewGraphicsSystem(gp, "test", rt)

	pl := sy.AddGraphicsPipeline("drawtri")
	// pl.SetFrontFace(wFrontFaceCCW)
	// pl.SetCullMode(wCullModeNone)
	// pl.SetAlphaBlend(false)

	sh := pl.AddShader("trianglelit")
	err = sh.OpenFile("testdata/trianglelit.wgsl")
	assert.NoError(t, err)
	pl.AddEntry(sh, VertexShader, "vs_main")
	pl.AddEntry(sh, FragmentShader, "fs_main")

	sy.Config()

	rt.CurrentFrame().ConfigReadBuffer()

	rp, err := sy.BeginRenderPass()
	assert.NoError(t, err)
	pl.BindPipeline(rp)
	rp.Draw(3, 1, 0, 0)
	rp.End()
	sy.AssertImage(t, rp, "triangle.png")
}
