// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"runtime"
	"time"
	"unsafe"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

//go:embed indexed.wgsl
var indexed string

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

type CamView struct {
	Model      math32.Matrix4
	View       math32.Matrix4
	Projection math32.Matrix4
}

func main() {
	gp := gpu.NewGPU()
	gpu.Debug = true
	gp.Config("drawidx")

	width, height := 1024, 768
	sp, terminate, pollEvents, err := gpu.GLFWCreateWindow(gp, width, height, "Draw Triangle Indexed")
	if err != nil {
		return
	}

	sf := gpu.NewSurface(gp, sp, width, height)

	fmt.Printf("format: %s\n", sf.Format.String())

	sy := gp.NewGraphicsSystem("drawidx", sf.Device)

	destroy := func() {
		sy.WaitDone()
		sy.Release()
		sf.Release()
		gp.Release()
		terminate()
	}

	pl := sy.AddGraphicsPipeline("drawidx")
	// sf.Format.SetMultisample(1)
	sy.ConfigRender(&sf.Format, gpu.Depth32)
	pl.SetCullMode(wgpu.CullModeNone)
	// sf.SetRender(&sy.Render)
	sy.SetClearColor(color.RGBA{50, 50, 50, 255})

	sh := pl.AddShader("indexed")
	sh.OpenCode(indexed)
	pl.AddEntry(sh, gpu.VertexShader, "vs_main")
	pl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	vgp := sy.Vars.AddVertexGroup()
	ugp := sy.Vars.AddGroup(gpu.Uniform)

	// vertex are dynamically sized in general, so using 0 here
	posv := vgp.Add("Pos", gpu.Float32Vector3, 0, gpu.VertexShader)
	clrv := vgp.Add("Color", gpu.Float32Vector3, 0, gpu.VertexShader)
	// note: index goes last usually
	idxv := vgp.Add("Index", gpu.Uint16, 0, gpu.VertexShader)
	idxv.Role = gpu.Index

	camv := ugp.AddStruct("Camera", int(unsafe.Sizeof(CamView{})), 1, gpu.VertexShader)

	vgp.SetNValues(1)
	ugp.SetNValues(1)
	sy.Config()

	triPos := posv.Values.Values[0]
	gpu.SetValueFrom(triPos, []float32{
		-0.5, 0.5, 0.0,
		0.5, 0.5, 0.0,
		0.0, -0.5, 0.0}) // negative point is UP in native Vulkan

	triClr := clrv.Values.Values[0]
	gpu.SetValueFrom(triClr, []float32{
		1.0, 0.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0})

	triIndex := idxv.Values.Values[0]
	gpu.SetValueFrom(triIndex, []uint16{0, 1, 2})
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
	aspect := float32(sf.Format.Size.X) / float32(sf.Format.Size.Y)
	fmt.Printf("aspect: %g\n", aspect)
	// VkPerspective version automatically flips Y axis and shifts depth
	// into a 0..1 range instead of -1..1, so original GL based geometry
	// will render identically here.
	camo.Projection.SetVkPerspective(45, aspect, 0.01, 100)
	gpu.SetValueFrom(cam, []CamView{camo}) // note: always use slice to copy

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		camo.Model.SetRotationY(.1 * float32(frameCount))
		gpu.SetValueFrom(cam, []CamView{camo})

		view, err := sf.AcquireNextTexture()
		if errors.Log(err) != nil {
			return
		}
		cmd := sy.NewCommandEncoder()
		rp := sy.BeginRenderPass(cmd, view)
		pl.BindPipeline(rp)
		pl.BindDrawVertex(rp)
		rp.End()
		sf.SubmitRender(cmd)
		sf.Present()

		frameCount++
		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 10 {
			fps := float64(frameCount) / dur
			fmt.Printf("fps: %.0f\n", fps)
			frameCount = 0
			stTime = eTime
		}
	}

	exitC := make(chan struct{}, 2)

	fpsDelay := time.Second / 60
	fpsTicker := time.NewTicker(fpsDelay)
	for {
		select {
		case <-exitC:
			fpsTicker.Stop()
			destroy()
			return
		case <-fpsTicker.C:
			if !pollEvents() {
				exitC <- struct{}{}
				continue
			}
			renderFrame()
		}
	}
}