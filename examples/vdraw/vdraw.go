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

	vk "github.com/vulkan-go/vulkan"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/mat32"
	"github.com/goki/vgpu/vdraw"
	"github.com/goki/vgpu/vgpu"
)

func init() {
	// must lock main thread for gpu!  this also means that vulkan must be used
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
	drw.ConfigSurface(sf)

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		drw.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		glfw.Terminate()
	}

	file, err := os.Open("teximg.jpg")
	if err != nil {
		fmt.Printf("image: %s\n", err)
	}
	tximg, _, err := image.Decode(file)
	file.Close()

	file, err = os.Open("ground.png")
	if err != nil {
		fmt.Printf("image: %s\n", err)
	}
	grimg, _, err := image.Decode(file)
	file.Close()

	file, err = os.Open("wood.png")
	if err != nil {
		fmt.Printf("image: %s\n", err)
	}
	wdimg, _, err := image.Decode(file)
	file.Close()

	_ = wdimg
	_ = tximg

	// multiple image array to avoid invalidating the descriptor set binding..
	// http://kylehalladay.com/blog/tutorial/vulkan/2018/01/28/Textue-Arrays-Vulkan.html

	drw.Start()
	drw.SetImage(grimg, vgpu.NoFlipY)
	drw.Scale(sf.Format.Bounds(), grimg.Bounds(), draw.Src)

	// drw.SetImage(tximg, vgpu.NoFlipY)
	drw.Copy(image.Point{40, 20}, grimg.Bounds(), draw.Src)

	// drw.SetImage(wdimg, vgpu.NoFlipY)
	drw.Copy(image.Point{600, 500}, grimg.Bounds(), draw.Src)

	// drw.FillRect(color.White, image.Rectangle{Min: image.Point{100, 80}, Max: image.Point{400, 200}}, draw.Src)
	// drw.FillRect(color.Black, image.Rectangle{Min: image.Point{500, 480}, Max: image.Point{400, 200}}, draw.Src)

	drw.End()

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
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

	fpsDelay := time.Second / 2
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
