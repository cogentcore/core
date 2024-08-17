// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"runtime"
	"time"
	"unsafe"

	vk "github.com/goki/vulkan"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/vgpu"
	"github.com/go-gl/glfw/v3.3/glfw"
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
	if vgpu.Init() != nil {
		return
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(1024, 768, "Draw Triangle", nil, nil)
	vgpu.IfPanic(err)

	// note: for graphics, require these instance extensions before init gpu!
	winext := window.GetRequiredInstanceExtensions()
	gp := vgpu.NewGPU()
	gp.AddInstanceExt(winext...)
	vgpu.Debug = true
	gp.Config("drawidx")

	// gp.PropertiesString(true) // print

	surfPtr, err := window.CreateWindowSurface(gp.Instance, nil)
	if err != nil {
		log.Println(err)
		return
	}
	sf := vgpu.NewSurface(gp, vk.SurfaceFromPointer(surfPtr))

	fmt.Printf("format: %s\n", sf.Format.String())

	sy := gp.NewGraphicsSystem("drawidx", &sf.Device)

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		sy.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		vgpu.Terminate()
	}

	pl := sy.NewPipeline("drawidx")
	// sf.Format.SetMultisample(1)
	sy.ConfigRender(&sf.Format, vgpu.Depth32)
	sf.SetRender(&sy.Render)
	sy.SetClearColor(0.2, 0.2, 0.2, 1)
	sy.SetRasterization(vk.PolygonModeFill, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)

	pl.AddShaderFile("indexed", vgpu.VertexShader, "indexed.spv")
	pl.AddShaderFile("vtxcolor", vgpu.FragmentShader, "vtxcolor.spv")

	vars := sy.Vars()
	vset := vars.AddVertexSet()
	set := vars.AddSet()

	nPts := 3

	posv := vset.Add("Pos", vgpu.Float32Vector3, nPts, vgpu.Vertex, vgpu.VertexShader)
	clrv := vset.Add("Color", vgpu.Float32Vector3, nPts, vgpu.Vertex, vgpu.VertexShader)
	// note: always put indexes last so there isn't a gap in the location indexes!
	// just the fact of adding one (and only one) Index type triggers indexed render
	idxv := vset.Add("Index", vgpu.Uint16, nPts, vgpu.Index, vgpu.VertexShader)

	camv := set.Add("Camera", vgpu.Struct, 1, vgpu.Uniform, vgpu.VertexShader)
	camv.SizeOf = vgpu.Float32Matrix4.Bytes() * 3 // no padding for these

	vset.ConfigValues(1) // one val per var
	set.ConfigValues(1)  // one val per var
	sy.Config()

	triPos, _ := posv.Values.ValueByIndexTry(0)
	triPosA := triPos.Floats32()
	triPosA.Set(0,
		-0.5, 0.5, 0.0,
		0.5, 0.5, 0.0,
		0.0, -0.5, 0.0) // negative point is UP in native Vulkan
	triPos.SetMod()

	triClr, _ := clrv.Values.ValueByIndexTry(0)
	triClrA := triClr.Floats32()
	triClrA.Set(0,
		1.0, 0.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0)
	triClr.SetMod()

	triIndex, _ := idxv.Values.ValueByIndexTry(0)
	idxs := []uint16{0, 1, 2}
	triIndex.CopyFromBytes(unsafe.Pointer(&idxs[0]))

	// This is the standard camera view projection computation
	cam, _ := camv.Values.ValueByIndexTry(0)
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
	camo.Projection.SetPerspective(45, aspect, 0.01, 100)

	cam.CopyFromBytes(unsafe.Pointer(&camo)) // sets mod

	sy.Mem.SyncToGPU()

	vars.BindDynamicValue(0, camv, cam)

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		camo.Model.SetRotationY(.1 * float32(frameCount))
		cam.CopyFromBytes(unsafe.Pointer(&camo)) // sets mod
		sy.Mem.SyncToGPU()

		idx, ok := sf.AcquireNextImage()
		if !ok {
			return
		}
		// fmt.Printf("\nacq: %v\n", time.Now().Sub(rt))
		descIndex := 0 // if running multiple frames in parallel, need diff sets
		cmd := sy.CmdPool.Buff
		sy.ResetBeginRenderPass(cmd, sf.Frames[idx], descIndex)
		pl.BindDrawVertex(cmd, descIndex)
		sy.EndRenderPass(cmd)
		// fmt.Printf("cmd %v\n", time.Now().Sub(rt))
		sf.SubmitRender(cmd) // this is where it waits for the 16 msec
		// fmt.Printf("submit %v\n", time.Now().Sub(rt))
		sf.PresentImage(idx)
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
