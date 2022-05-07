// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"runtime"
	"time"
	"unsafe"

	vk "github.com/vulkan-go/vulkan"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
)

func init() {
	// a must lock main thread for gpu!  this also means that vulkan must be used
	// for gogi/oswin eventually if we want gui and compute
	runtime.LockOSThread()
}

var TheGPU *vgpu.GPU

type CamView struct {
	Model mat32.Mat4
	View  mat32.Mat4
	Prjn  mat32.Mat4
}

func main() {
	glfw.Init()
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	vk.Init()

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(1024, 768, "Draw Texture", nil, nil)
	vgpu.IfPanic(err)

	// note: for graphics, require these instance extensions before init gpu!
	winext := window.GetRequiredInstanceExtensions()
	gp := vgpu.NewGPU()
	gp.AddInstanceExt(winext...)
	gp.Debug = true
	gp.Config("texture")
	TheGPU = gp

	// gp.PropsString(true) // print

	surfPtr, err := window.CreateWindowSurface(gp.Instance, nil)
	if err != nil {
		log.Println(err)
		return
	}
	sf := vgpu.NewSurface(gp, vk.SurfaceFromPointer(surfPtr))

	fmt.Printf("format: %s\n", sf.Format.String())

	sy := gp.NewGraphicsSystem("texture", &sf.Device)
	pl := sy.NewPipeline("texture")

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		sy.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		glfw.Terminate()
	}

	sy.ConfigRenderPass(&sf.Format, vgpu.Depth32)
	sf.SetRenderPass(&sy.RenderPass)
	pl.SetGraphicsDefaults()
	pl.SetClearColor(0.2, 0.2, 0.2, 1)
	pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)

	pl.AddShaderFile("texture_vert", vgpu.VertexShader, "texture_vert.spv")
	pl.AddShaderFile("texture_frag", vgpu.FragmentShader, "texture_frag.spv")

	posv := sy.Vars.Add("Pos", vgpu.Float32Vec3, vgpu.Vertex, 0, vgpu.VertexShader)
	clrv := sy.Vars.Add("Color", vgpu.Float32Vec3, vgpu.Vertex, 0, vgpu.VertexShader)
	txcv := sy.Vars.Add("TexCoord", vgpu.Float32Vec2, vgpu.Vertex, 0, vgpu.VertexShader)
	// note: always put indexes last so there isn't a gap in the location indexes!
	idxv := sy.Vars.Add("Index", vgpu.Uint16, vgpu.Index, 0, vgpu.VertexShader)

	camv := sy.Vars.Add("Camera", vgpu.Struct, vgpu.Uniform, 0, vgpu.VertexShader)
	camv.SizeOf = vgpu.Float32Mat4.Bytes() * 3 // no padding for these

	tximgv := sy.Vars.Add("TexSampler", vgpu.ImageRGBA32, vgpu.TextureRole, 0, vgpu.FragmentShader)

	nPts := 4
	nIdxs := 6
	sqrPos := sy.Mem.Vals.Add("SqrPos", posv, nPts)
	sqrClr := sy.Mem.Vals.Add("SqrClr", clrv, nPts)
	sqrTex := sy.Mem.Vals.Add("SqrTex", txcv, nPts)
	sqrIdx := sy.Mem.Vals.Add("SqrIdx", idxv, nIdxs)
	sqrPos.Indexes = "SqrIdx" // only need to set indexes for one vertex val
	cam := sy.Mem.Vals.Add("Camera", camv, 1)

	img := sy.Mem.Vals.Add("TexImage", tximgv, 1)
	file, err := os.Open("teximg.jpg")
	if err != nil {
		fmt.Printf("image: %s\n", err)
	}
	gimg, _, err := image.Decode(file)
	file.Close()
	img.Texture.ConfigGoImage(gimg)
	// img.Texture.Sampler.Border = vgpu.BorderBlack
	// img.Texture.Sampler.UMode = vgpu.ClampToBorder
	// img.Texture.Sampler.VMode = vgpu.ClampToBorder

	// note: add all values per above before doing Config
	sy.Config()
	sy.Mem.Config()

	// note: first val in set is offset
	sqrPosA := sqrPos.Floats32()
	sqrPosA.Set(0,
		-0.5, -0.5, 0.0,
		0.5, -0.5, 0.0,
		0.5, 0.5, 0.0,
		-0.5, 0.5, 0.0)
	sqrPos.Mod = true

	sqrClrA := sqrClr.Floats32()
	sqrClrA.Set(0,
		1.0, 0.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0,
		1.0, 1.0, 0.0)
	sqrClr.Mod = true

	sqrTexA := sqrTex.Floats32()
	sqrTexA.Set(0,
		1.0, 0.0,
		0.0, 0.0,
		0.0, 1.0,
		1.0, 1.0)
	sqrTex.Mod = true

	idxs := []uint16{0, 1, 2, 0, 2, 3}
	sqrIdx.CopyBytes(unsafe.Pointer(&idxs[0]))

	img.SetGoImage(gimg, vgpu.FlipY)

	// This is the standard camera view projection computation
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
	aspect := float32(sf.Format.Size.X) / float32(sf.Format.Size.Y)
	// fmt.Printf("aspect: %g\n", aspect)
	// VkPerspective version automatically flips Y axis and shifts depth
	// into a 0..1 range instead of -1..1, so original GL based geometry
	// will render identically here.
	camo.Prjn.SetVkPerspective(45, aspect, 0.01, 100)

	cam.CopyBytes(unsafe.Pointer(&camo)) // sets mod

	sy.Mem.SyncToGPU()

	sy.SetVals(0, "SqrPos", "SqrClr", "SqrTex", "Camera", "TexImage")

	if sy.Vars.Validate() != nil {
		destroy()
		return
	}

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		camo.Model.SetRotationY(.002 * float32(frameCount))
		cam.CopyBytes(unsafe.Pointer(&camo)) // sets mod
		sy.Mem.SyncToGPU()

		idx := sf.AcquireNextImage()
		// fmt.Printf("\nacq: %v\n", time.Now().Sub(rt))
		pl.FullStdRender(pl.CmdPool.Buff, sf.Frames[idx])
		// fmt.Printf("cmd %v\n", time.Now().Sub(rt))
		sf.SubmitRender(pl.CmdPool.Buff) // this is where it waits for the 16 msec
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
