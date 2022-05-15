// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/goki/mat32"
	vk "github.com/goki/vulkan"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vgpu/vgpu"
	"github.com/goki/vgpu/vphong"
)

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
	window, err := glfw.CreateWindow(1024, 768, "vPhong Test", nil, nil)
	vgpu.IfPanic(err)

	// note: for graphics, require these instance extensions before init gpu!
	winext := window.GetRequiredInstanceExtensions()
	gp := vgpu.NewGPU()
	gp.AddInstanceExt(winext...)
	gp.Debug = true
	gp.Config("vPhong test")
	TheGPU = gp

	// gp.PropsString(true) // print

	surfPtr, err := window.CreateWindowSurface(gp.Instance, nil)
	if err != nil {
		log.Println(err)
		return
	}
	sf := vgpu.NewSurface(gp, vk.SurfaceFromPointer(surfPtr))

	fmt.Printf("format: %s\n", sf.Format.String())

	ph := &vphong.Phong{}
	sy := &ph.Sys
	sy.InitGraphics(sf.GPU, "vphong.Phong", &sf.Device)
	sy.ConfigRenderPass(&sf.Format, vgpu.Depth32)
	sf.SetRenderPass(&sy.RenderPass)
	ph.ConfigSys()

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		ph.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		glfw.Terminate()
	}

	/////////////////////////////
	// Lights

	dark := color.RGBA{50, 50, 50, 50}
	ph.AddAmbientLight(vphong.NewGoColor(dark))
	ph.AddDirLight(vphong.NewGoColor(color.White), mat32.Vec3{0, 0, -1})

	/////////////////////////////
	// Meshes

	ph.AddMesh("rect", 4, 6, false)

	imgFiles := []string{"ground.png", "wood.png", "teximg.jpg"}
	imgs := make([]image.Image, len(imgFiles))
	for i, fnm := range imgFiles {
		imgs[i] = OpenImage(fnm)
		ph.AddTexture(fnm, vphong.NewTexture(imgs[i], mat32.Vec2{1, 1}, mat32.Vec2{0, 0}))
	}

	/////////////////////////////
	// Colors

	blue := color.RGBA{0, 0, 255, 255}
	ph.AddColor("blue", vphong.NewColors(blue, color.Black, color.White, 30, 1))

	/////////////////////////////
	// Camera / Mtxs

	// This is the standard camera view projection computation
	campos := mat32.Vec3{0, 0, 2}
	target := mat32.Vec3{0, 0, 0}
	var lookq mat32.Quat
	lookq.SetFromRotationMatrix(mat32.NewLookAt(campos, target, mat32.Vec3Y))
	scale := mat32.Vec3{1, 1, 1}
	var cview mat32.Mat4
	cview.SetTransform(campos, lookq, scale)
	view, _ := cview.Inverse()

	var model mat32.Mat4
	model.SetIdentity()
	model.SetRotationY(.3)

	aspect := float32(sf.Format.Size.X) / float32(sf.Format.Size.Y)
	// fmt.Printf("aspect: %g\n", aspect)
	// VkPerspective version automatically flips Y axis and shifts depth
	// into a 0..1 range instead of -1..1, so original GL based geometry
	// will render identically here.

	var prjn mat32.Mat4
	prjn.SetVkPerspective(45, aspect, 0.01, 100)

	ph.AddMtxs("mtx1", vphong.NewMtxs(&model, view, &prjn))

	/////////////////////////////
	//  Config!

	ph.Config()

	/////////////////////////////
	//  Set Mesh values

	pos, norm, tex, _, idx := ph.MeshFloatsByName("rect")
	pos.Set(0,
		-0.5, -0.5, 0.0,
		0.5, -0.5, 0.0,
		0.5, 0.5, 0.0,
		-0.5, 0.5, 0.0)

	norm.Set(0,
		0.0, 0.0, 1.0,
		0.0, 0.0, 1.0,
		0.0, 0.0, 1.0,
		0.0, 0.0, 1.0)

	tex.Set(0,
		1.0, 0.0,
		0.0, 0.0,
		0.0, 1.0,
		1.0, 1.0)

	idx.Set(0, 0, 1, 2, 0, 2, 3)

	ph.ModMeshByName("rect")

	ph.Sync()

	render1 := func() {
		ph.UseColorName("blue")
		ph.UseMtxsName("mtx1")
		ph.UseMeshName("rect")
		ph.UseTextureName("teximg.jpg")
		ph.Render()
	}

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		idx := sf.AcquireNextImage()
		cmd := sy.CmdPool.Buff
		descIdx := 0 // if running multiple frames in parallel, need diff sets
		sy.ResetBeginRenderPass(cmd, sf.Frames[idx], descIdx)

		fcr := frameCount % 10
		_ = fcr

		render1()

		// switch {
		// case fcr < 3:
		// 	scaleImg(fcr)
		// case fcr < 6:
		// 	copyImg(fcr - 3)
		// default:
		// 	fillRnd()
		// }
		frameCount++

		sy.EndRenderPass(cmd)

		sf.SubmitRender(cmd) // this is where it waits for the 16 msec
		sf.PresentImage(idx)

		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 100 {
			fps := float64(frameCount) / dur
			fmt.Printf("fps: %.0f\n", fps)
			frameCount = 0
			stTime = eTime
		}
	}

	glfw.PollEvents()
	renderFrame()
	glfw.PollEvents()

	exitC := make(chan struct{}, 2)

	fpsDelay := time.Second / 5
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
