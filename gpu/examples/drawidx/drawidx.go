// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image/color"
	"log/slog"
	"runtime"
	"time"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/rajveermalviya/go-webgpu/wgpu"
	wgpuext_glfw "github.com/rajveermalviya/go-webgpu/wgpuext/glfw"
)

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
	if gpu.Init() != nil {
		return
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(1024, 768, "Draw Triangle", nil, nil)
	if err != nil {
		panic(err)
	}

	gp := gpu.NewGPU()
	gpu.Debug = true
	gp.Config("drawidx")

	sp := gp.Instance.CreateSurface(wgpuext_glfw.GetSurfaceDescriptor(window))
	width, height := window.GetSize()
	sf := gpu.NewSurface(gp, sp, width, height)

	fmt.Printf("format: %s\n", sf.Format.String())

	sy := gp.NewGraphicsSystem("drawidx", sf.Device)

	destroy := func() {
		// vk.DeviceWaitIdle(sf.Device.Device)
		sy.Release()
		sf.Release()
		gp.Release()
		window.Destroy()
		gpu.Terminate()
	}

	pl := sy.AddGraphicsPipeline("drawidx")
	// sf.Format.SetMultisample(1)
	sy.ConfigRender(&sf.Format, gpu.Depth32)
	pl.SetCullMode(wgpu.CullModeNone)
	// sf.SetRender(&sy.Render)
	sy.SetClearColor(color.RGBA{50, 50, 50, 255})

	sh := pl.AddShader("indexed")
	sh.OpenFile("indexed.wgsl")
	pl.AddEntry(sh, gpu.VertexShader, "vs_main")
	pl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	vgp := sy.Vars.AddVertexGroup()
	ugp := sy.Vars.AddGroup(gpu.Uniform)

	nPts := 3

	posv := vgp.Add("Pos", gpu.Float32Vector3, nPts, gpu.VertexShader)
	clrv := vgp.Add("Color", gpu.Float32Vector3, nPts, gpu.VertexShader)
	// note: always put indexes last so there isn't a gap in the location indexes!
	// just the fact of adding one (and only one) Index type triggers indexed render
	idxv := vgp.Add("Index", gpu.Uint16, nPts, gpu.VertexShader)
	idxv.Role = gpu.Index

	camv := ugp.AddStruct("Camera", gpu.Float32Matrix4.Bytes()*3, 1, gpu.VertexShader)

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

	// if sy.Validate() != nil { // useful check, makes any read-out buffers
	// 	return
	// }

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		camo.Model.SetRotationY(.1 * float32(frameCount))
		gpu.SetValueFrom(cam, []CamView{camo})

		view, err := sf.AcquireNextTexture()
		if err != nil {
			slog.Error(err.Error())
			return
		}
		// fmt.Printf("\nacq: %v\n", time.Now().Sub(rt))
		cmd := sy.NewCommandEncoder()
		rp := sy.BeginRenderPass(cmd, view)
		pl.BindPipeline(rp)
		pl.BindDrawVertex(rp)
		rp.End()
		// fmt.Printf("cmd %v\n", time.Now().Sub(rt))
		sf.SubmitRender(cmd) // this is where it waits for the 16 msec
		// fmt.Printf("submit %v\n", time.Now().Sub(rt))
		sf.Present()
		// fmt.Printf("present %v\n\n", time.Now().Sub(rt))
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
			if window.ShouldClose() {
				exitC <- struct{}{}
				continue
			}
			glfw.PollEvents()
			renderFrame()
		}
	}
}
