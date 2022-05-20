// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"runtime"
	"time"
	"unsafe"

	"github.com/goki/mat32"
	vk "github.com/goki/vulkan"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vgpu/vdraw"
	"github.com/goki/vgpu/vgpu"
)

type CamView struct {
	Model mat32.Mat4
	View  mat32.Mat4
	Prjn  mat32.Mat4
}

func init() {
	// must lock main thread for gpu!  this also means that vulkan must be used
	// for gogi/oswin eventually if we want gui and compute
	runtime.LockOSThread()
}

var TheGPU *vgpu.GPU

func OpenImage(fname string) image.Image {
	file, err := os.Open(fname)
	defer file.Close()
	if err != nil {
		fmt.Printf("image: %s\n", err)
	}
	gimg, _, err := image.Decode(file)
	if err != nil {
		fmt.Println(err)
	}
	return gimg
}

func main() {
	glfw.Init()
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	vk.Init()

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(1024, 768, "vDraw Test", nil, nil)
	vgpu.IfPanic(err)

	// note: for graphics, require these instance extensions before init gpu!
	winext := window.GetRequiredInstanceExtensions()
	gp := vgpu.NewGPU()
	gp.AddInstanceExt(winext...)
	gp.Debug = true
	gp.Config("vDraw test")
	TheGPU = gp

	// gp.PropsString(true) // print

	surfPtr, err := window.CreateWindowSurface(gp.Instance, nil)
	if err != nil {
		log.Println(err)
		return
	}
	sf := vgpu.NewSurface(gp, vk.SurfaceFromPointer(surfPtr))

	fmt.Printf("format: %s\n", sf.Format.String())

	drw := &vdraw.Drawer{}
	sf.Format.SetMultisample(1)
	drw.ConfigSurface(sf, 10, 10) // 10 = max number of images, colors to choose for rendering

	rf := vgpu.NewRenderFrame(gp, &sf.Device, image.Point{1024, 768})
	sy := gp.NewGraphicsSystem("drawidx", &rf.Device)

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		sy.Destroy()
		rf.Destroy()
		drw.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		glfw.Terminate()
	}

	/////////////////////////////////
	// RenderFrame

	pl := sy.NewPipeline("drawidx")
	sy.ConfigRenderNonSurface(&rf.Format, vgpu.Depth32) // not surface = renderframe
	rf.SetRender(&sy.Render)
	sy.SetClearColor(0.2, 0.2, 0.2, 1)
	sy.SetRasterization(vk.PolygonModeFill, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)

	pl.AddShaderFile("indexed", vgpu.VertexShader, "indexed.spv")
	pl.AddShaderFile("vtxcolor", vgpu.FragmentShader, "vtxcolor.spv")

	vars := sy.Vars()
	vset := vars.AddVertexSet()
	set := vars.AddSet()

	nPts := 3

	posv := vset.Add("Pos", vgpu.Float32Vec3, nPts, vgpu.Vertex, vgpu.VertexShader)
	clrv := vset.Add("Color", vgpu.Float32Vec3, nPts, vgpu.Vertex, vgpu.VertexShader)
	// note: always put indexes last so there isn't a gap in the location indexes!
	// just the fact of adding one (and only one) Index type triggers indexed render
	idxv := vset.Add("Index", vgpu.Uint16, nPts, vgpu.Index, vgpu.VertexShader)

	camv := set.Add("Camera", vgpu.Struct, 1, vgpu.Uniform, vgpu.VertexShader)
	camv.SizeOf = vgpu.Float32Mat4.Bytes() * 3 // no padding for these

	vset.ConfigVals(1) // one val per var
	set.ConfigVals(1)  // one val per var
	sy.Config()

	triPos, _ := posv.Vals.ValByIdxTry(0)
	triPosA := triPos.Floats32()
	triPosA.Set(0,
		-0.5, 0.5, 0.0,
		0.5, 0.5, 0.0,
		0.0, -0.5, 0.0) // negative point is UP in native Vulkan
	triPos.SetMod()

	triClr, _ := clrv.Vals.ValByIdxTry(0)
	triClrA := triClr.Floats32()
	triClrA.Set(0,
		1.0, 0.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0)
	triClr.SetMod()

	triIdx, _ := idxv.Vals.ValByIdxTry(0)
	idxs := []uint16{0, 1, 2}
	triIdx.CopyBytes(unsafe.Pointer(&idxs[0]))

	// This is the standard camera view projection computation
	cam, _ := camv.Vals.ValByIdxTry(0)
	campos := mat32.Vec3{0, 0, 2}
	target := mat32.Vec3{0, 0, 0}
	var lookq mat32.Quat
	lookq.SetFromRotationMatrix(mat32.NewLookAt(campos, target, mat32.Vec3Y))
	scale := mat32.Vec3{1, 1, 1}
	var cview mat32.Mat4
	cview.SetTransform(campos, lookq, scale)
	view, _ := cview.Inverse()

	var camo CamView
	camo.Model.SetIdentity()
	camo.View.CopyFrom(view)

	updateAspect := func() {
		aspect := rf.Format.Aspect()
		fmt.Printf("aspect: %g\n", aspect)
		camo.Prjn.SetVkPerspective(45, aspect, 0.01, 100)
		cam.CopyBytes(unsafe.Pointer(&camo)) // sets mod
		sy.Mem.SyncToGPU()
	}

	updateAspect()

	vars.BindDynVal(0, camv, cam)

	drw.ConfigImage(0, &rf.Format)

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		camo.Model.SetRotationY(.1 * float32(frameCount))
		cam.CopyBytes(unsafe.Pointer(&camo)) // sets mod
		sy.Mem.SyncToGPU()

		idx := 0 // sf.AcquireNextImage()
		// fmt.Printf("\nacq: %v\n", time.Now().Sub(rt))
		descIdx := 0 // if running multiple frames in parallel, need diff sets

		fr := rf.Frames[idx]
		cmd := sy.CmdPool.Buff
		sy.ResetBeginRenderPass(cmd, fr, descIdx)
		pl.BindDrawVertex(cmd, descIdx)
		sy.EndRenderPass(cmd)
		rf.SubmitRender(cmd) // this is where it waits for the 16 msec
		rf.WaitForRender()

		if false { // grab
			// tcmd := sy.MemCmdStart()
			// rf.GrabImage(tcmd, 0)
			// sy.MemCmdSubmitWaitFree()
			// gimg, err := fr.Render.Grab.DevGoImage()
			// if err == nil {
			// 	gi.SaveImage("render.png", gimg)
			// } else {
			// 	fmt.Printf("image grab err: %s\n", err)
			// }
		}

		drw.SetFrameImage(0, fr)
		drw.SyncImages()
		drw.StartDraw()
		drw.Scale(0, sf.Format.Bounds(), image.ZR, draw.Src)
		drw.EndDraw()

		// fmt.Printf("present %v\n\n", time.Now().Sub(rt))
		frameCount++
		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 10 {
			fps := float64(frameCount) / dur
			fmt.Printf("fps: %.0f\n", fps)
			sz := rf.Format.Size
			sz.X -= 10
			rf.SetSize(sz)
			updateAspect()
			frameCount = 0
			stTime = eTime
		}
	}

	glfw.PollEvents()
	renderFrame()
	glfw.PollEvents()

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
