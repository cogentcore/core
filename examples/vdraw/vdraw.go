// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	vk "github.com/goki/vulkan"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vgpu/vdraw"
	"github.com/goki/vgpu/vgpu"
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
	drw.ConfigSurface(sf, 10) // 10 = max number of colors or images to choose for rendering

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		drw.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		glfw.Terminate()
	}

	imgFiles := []string{"ground.png", "wood.png", "teximg.jpg"}
	imgs := make([]image.Image, len(imgFiles))
	for i, fnm := range imgFiles {
		imgs[i] = OpenImage(fnm)
	}

	scaleImg := func(idx int) {
		drw.SetImage(imgs[idx], vgpu.NoFlipY)
		drw.Scale(sf.Format.Bounds(), imgs[idx].Bounds(), draw.Src)
	}
	copyImg := func(idx int) {
		drw.SetImage(imgs[idx], vgpu.NoFlipY)
		drw.Copy(image.Point{rand.Intn(500), rand.Intn(500)}, imgs[idx].Bounds(), draw.Src)
	}

	_ = scaleImg
	_ = copyImg

	pal := vdraw.Palette{}
	pal.Add("white", color.White)
	pal.Add("black", color.Black)
	pal.Add("red", color.RGBA{255, 0, 0, 255})
	pal.Add("green", color.RGBA{0, 255, 0, 255})
	pal.Add("blue", color.RGBA{0, 0, 255, 255})

	drw.SetPalette(pal)

	fillRnd := func() {
		nclr := len(pal)
		drw.StartFill()
		for i := 0; i < 5; i++ {
			sp := image.Point{rand.Intn(500), rand.Intn(500)}
			sz := image.Point{rand.Intn(500), rand.Intn(500)}
			drw.FillRect(i%nclr, image.Rectangle{Min: sp, Max: sp.Add(sz)}, draw.Src)
		}
		drw.EndFill()
	}

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		fcr := frameCount % 10
		_ = fcr
		switch {
		case fcr < 3:
			scaleImg(fcr)
		case fcr < 6:
			copyImg(fcr - 3)
		default:
			fillRnd()
		}
		frameCount++

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

	fpsDelay := 2 * time.Second
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
