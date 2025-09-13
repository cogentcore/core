// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"
	"image/color"
	"testing"
	"unsafe"

	_ "cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/math32"
	"github.com/cogentcore/webgpu/wgpu"
	"github.com/stretchr/testify/assert"
)

type CamView struct {
	Model      math32.Matrix4
	View       math32.Matrix4
	Projection math32.Matrix4
}

func TestGPUTriangle(t *testing.T) {
	t.Skip("Need software GPU on CI")
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

func TestGPUIndexed(t *testing.T) {
	t.Skip("Need software GPU on CI")
	gp, dev, err := NoDisplayGPU()
	assert.NoError(t, err)
	sz := image.Point{480, 320}
	rt := NewRenderTexture(gp, dev, sz, 4, Depth32)
	sy := NewGraphicsSystem(gp, "test", rt)

	pl := sy.AddGraphicsPipeline("drawtri")
	pl.SetCullMode(wgpu.CullModeNone)
	sy.SetClearColor(color.RGBA{50, 50, 50, 255})

	sh := pl.AddShader("indexed")
	err = sh.OpenFile("testdata/indexed.wgsl")
	assert.NoError(t, err)
	pl.AddEntry(sh, VertexShader, "vs_main")
	pl.AddEntry(sh, FragmentShader, "fs_main")

	vgp := sy.Vars().AddVertexGroup()
	ugp := sy.Vars().AddGroup(Uniform)

	// vertex are dynamically sized in general, so using 0 here
	posv := vgp.Add("Pos", Float32Vector3, 0, VertexShader)
	clrv := vgp.Add("Color", Float32Vector3, 0, VertexShader)
	// note: index goes last usually
	idxv := vgp.Add("Index", Uint16, 0, VertexShader)
	idxv.Role = Index

	camv := ugp.AddStruct("Camera", int(unsafe.Sizeof(CamView{})), 1, VertexShader)

	vgp.SetNValues(1)
	ugp.SetNValues(1)
	sy.Config()

	triPos := posv.Values.Values[0]
	SetValueFrom(triPos, []float32{
		-0.5, 0.5, 0.0,
		0.5, 0.5, 0.0,
		0.0, -0.5, 0.0}) // negative point is UP in native Vulkan

	triClr := clrv.Values.Values[0]
	SetValueFrom(triClr, []float32{
		1.0, 0.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0})

	triIndex := idxv.Values.Values[0]
	SetValueFrom(triIndex, []uint16{0, 1, 2})
	// note: the only way to set indexes is at start..

	// This is the standard camera view projection computation
	cam := camv.Values.Values[0]
	campos := math32.Vec3(0, 0, 2)
	target := math32.Vec3(0, 0, 0)
	var lookq math32.Quat
	lookq.SetFromRotationMatrix(math32.NewLookAt(campos, target, math32.Vec3(0, 1, 0)))
	scale := math32.Vec3(1, 1, 1)
	var cview math32.Matrix4
	cview.SetTransform(campos, lookq, scale)
	view, _ := cview.Inverse()

	var camo CamView
	camo.Model.SetIdentity()
	camo.View.CopyFrom(view)
	aspect := float32(rt.Format.Size.X) / float32(rt.Format.Size.Y)
	// fmt.Printf("aspect: %g\n", aspect)
	camo.Projection.SetPerspective(45, aspect, 0.01, 100)
	SetValueFrom(cam, []CamView{camo}) // note: always use slice to copy

	rt.CurrentFrame().ConfigReadBuffer()

	rp, err := sy.BeginRenderPass()
	assert.NoError(t, err)
	pl.BindPipeline(rp)
	pl.BindDrawIndexed(rp)
	rp.End()
	sy.AssertImage(t, rp, "indexed.png")
}
