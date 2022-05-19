// Copyright (c) 2022, The GoKi Authors. All rights reserved.
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

	vk "github.com/goki/vulkan"

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

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		sy.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		glfw.Terminate()
	}

	pl := sy.NewPipeline("texture")
	sy.ConfigRender(&sf.Format, vgpu.Depth32)
	sf.SetRender(&sy.Render)
	sy.SetClearColor(0.2, 0.2, 0.2, 1)
	sy.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)

	pl.AddShaderFile("texture_vert", vgpu.VertexShader, "texture_vert.spv")
	pl.AddShaderFile("texture_frag", vgpu.FragmentShader, "texture_frag.spv")

	vars := sy.Vars()
	vset := vars.AddVertexSet()
	pcset := vars.AddPushSet()
	uset := vars.AddSet()
	txset := vars.AddSet()

	nPts := 4
	nIdxs := 6

	posv := vset.Add("Pos", vgpu.Float32Vec3, nPts, vgpu.Vertex, vgpu.VertexShader)
	clrv := vset.Add("Color", vgpu.Float32Vec3, nPts, vgpu.Vertex, vgpu.VertexShader)
	txcv := vset.Add("TexCoord", vgpu.Float32Vec2, nPts, vgpu.Vertex, vgpu.VertexShader)
	// note: always put indexes last so there isn't a gap in the location indexes!
	idxv := vset.Add("Index", vgpu.Uint16, nIdxs, vgpu.Index, vgpu.VertexShader)

	camv := uset.AddStruct("Camera", vgpu.Float32Mat4.Bytes()*3, 1, vgpu.Uniform, vgpu.VertexShader)

	txidxv := pcset.Add("TexIdx", vgpu.Int32, 1, vgpu.Push, vgpu.FragmentShader)
	tximgv := txset.Add("TexSampler", vgpu.ImageRGBA32, 1, vgpu.TextureRole, vgpu.FragmentShader)

	vset.ConfigVals(1) // val per var
	uset.ConfigVals(1)
	txset.ConfigVals(3)

	imgFiles := []string{"ground.png", "wood.png", "teximg.jpg"}
	imgs := make([]image.Image, len(imgFiles))
	for i, fnm := range imgFiles {
		imgs[i] = OpenImage(fnm)
		img, _ := tximgv.Vals.ValByIdxTry(i)
		img.Texture.ConfigGoImage(imgs[i])
		// img.Texture.Sampler.Border = vgpu.BorderBlack
		// img.Texture.Sampler.UMode = vgpu.ClampToBorder
		// img.Texture.Sampler.VMode = vgpu.ClampToBorder
	}

	sy.Config() // allocates everything etc

	// note: first val in set is offset
	rectPos, _ := posv.Vals.ValByIdxTry(0)
	rectPosA := rectPos.Floats32()
	rectPosA.Set(0,
		-0.5, -0.5, 0.0,
		0.5, -0.5, 0.0,
		0.5, 0.5, 0.0,
		-0.5, 0.5, 0.0)
	rectPos.SetMod()

	rectClr, _ := clrv.Vals.ValByIdxTry(0)
	rectClrA := rectClr.Floats32()
	rectClrA.Set(0,
		1.0, 0.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0,
		1.0, 1.0, 0.0)
	rectClr.SetMod()

	rectTex, _ := txcv.Vals.ValByIdxTry(0)
	rectTexA := rectTex.Floats32()
	rectTexA.Set(0,
		1.0, 0.0,
		0.0, 0.0,
		0.0, 1.0,
		1.0, 1.0)
	rectTex.SetMod()

	rectIdx, _ := idxv.Vals.ValByIdxTry(0)
	idxs := []uint16{0, 1, 2, 0, 2, 3}
	rectIdx.CopyBytes(unsafe.Pointer(&idxs[0]))

	for i, gimg := range imgs {
		img, _ := tximgv.Vals.ValByIdxTry(i)
		img.SetGoImage(gimg, vgpu.FlipY)
	}

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
	aspect := float32(sf.Format.Size.X) / float32(sf.Format.Size.Y)
	// fmt.Printf("aspect: %g\n", aspect)
	// VkPerspective version automatically flips Y axis and shifts depth
	// into a 0..1 range instead of -1..1, so original GL based geometry
	// will render identically here.
	camo.Prjn.SetVkPerspective(45, aspect, 0.01, 100)

	cam.CopyBytes(unsafe.Pointer(&camo)) // sets mod

	sy.Mem.SyncToGPU()

	vars.BindVarsStart(0) // only one set of bindings
	vars.BindStatVars(1)  // gets images
	vars.BindVarsEnd()

	vars.BindDynVal(0, camv, cam)

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		camo.Model.SetRotationY(.002 * float32(frameCount))
		cam.CopyBytes(unsafe.Pointer(&camo)) // sets mod
		sy.Mem.SyncToGPU()

		imgIdx := int32(frameCount % len(imgs))

		idx := sf.AcquireNextImage()

		cmd := sy.CmdPool.Buff
		descIdx := 0 // if running multiple frames in parallel, need diff sets

		sy.ResetBeginRenderPass(cmd, sf.Frames[idx], descIdx)
		pl.Push(cmd, txidxv, vgpu.FragmentShader, unsafe.Pointer(&imgIdx))
		pl.BindDrawVertex(cmd, descIdx)
		sy.EndRenderPass(cmd)

		sf.SubmitRender(cmd) // this is where it waits for the 16 msec
		sf.PresentImage(idx)
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

	fpsDelay := time.Second / 10
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
