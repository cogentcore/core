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
	"path/filepath"
	"runtime"
	"time"

	vk "github.com/goki/vulkan"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vgpu/vdraw"
	"github.com/goki/vgpu/vgpu"
)

func init() {
	// must lock main thread for gpu!
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
	if vgpu.Init() != nil {
		return
	}

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
	drw.YIsDown = true
	drw.ConfigSurface(sf, 16) // requires 2 NDesc

	drw.SetMaxTextures(32) // test resizing

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		drw.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		vgpu.Terminate()
	}

	imgFiles := []string{"ground.png", "wood.png", "teximg.jpg"}
	imgs := make([]image.Image, len(imgFiles))

	stoff := 15 // causes images to wrap around sets, so this tests that..

	for i, fnm := range imgFiles {
		pnm := filepath.Join("../images", fnm)
		imgs[i] = OpenImage(pnm)
		drw.SetGoImage(i+stoff, 0, imgs[i], vgpu.NoFlipY)
	}

	// icons loaded into a texture array
	iconFiles := []string{"sound1.png", "text.png", "up.png", "world1.png"}
	iconImgs := make([]image.Image, len(iconFiles))
	iconIdx := 0
	iconFmt := vgpu.NewImageFormat(20, 22, len(iconFiles))
	drw.ConfigImage(iconIdx, iconFmt)
	for i, fnm := range iconFiles {
		pnm := filepath.Join("../images", fnm)
		iconImgs[i] = OpenImage(pnm)
		drw.SetGoImage(iconIdx, i, iconImgs[i], vgpu.NoFlipY)
	}

	drw.SyncImages()

	rendImgs := func(idx int) {
		descIdx := 0
		if idx+stoff >= vgpu.MaxTexturesPerSet {
			descIdx = 1
		}
		drw.StartDraw(descIdx) // specifically starting with correct descIdx is key..
		drw.Scale(idx+stoff, 0, sf.Format.Bounds(), image.ZR, vdraw.Src, vgpu.NoFlipY)
		for i := range imgFiles {
			// dp := image.Point{rand.Intn(500), rand.Intn(500)}
			dp := image.Point{i * 50, i * 50}
			drw.Copy(i+stoff, 0, dp, image.ZR, vdraw.Src, vgpu.NoFlipY)
		}
		for i := range iconFiles {
			dp := image.Point{rand.Intn(500), rand.Intn(500)}
			drw.Copy(iconIdx, i, dp, image.ZR, vdraw.Over, vgpu.NoFlipY)
		}
		drw.EndDraw()
	}

	_ = rendImgs

	red := color.RGBA{255, 0, 0, 255}
	green := color.RGBA{0, 255, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}

	colors := []color.Color{color.White, color.Black, red, green, blue}

	fillRnd := func() {
		nclr := len(colors)
		drw.StartFill()
		for i := 0; i < 5; i++ {
			sp := image.Point{rand.Intn(500), rand.Intn(500)}
			sz := image.Point{rand.Intn(500), rand.Intn(500)}
			drw.FillRect(colors[i%nclr], image.Rectangle{Min: sp, Max: sp.Add(sz)}, draw.Src)
		}
		drw.EndFill()
	}

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		fcr := frameCount % 4
		_ = fcr
		switch {
		case fcr < 3:
			rendImgs(fcr)
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

	fpsDelay := time.Second / 1
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
