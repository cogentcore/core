// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math/rand"
	"runtime"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/examples/images"
	"cogentcore.org/core/gpu/gpudraw"
)

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	gp := gpu.NewGPU()
	gpu.Debug = true
	gp.Config("gpudraw")

	width, height := 1024, 768
	sp, terminate, pollEvents, err := gpu.GLFWCreateWindow(gp, width, height, "GPU Draw")
	if err != nil {
		return
	}

	sf := gpu.NewSurface(gp, sp, width, height)

	drw := gpudraw.NewDrawerSurface(sf)

	destroy := func() {
		drw.Release()
		sf.Release()
		gp.Release()
		terminate()
	}

	imgFiles := []string{"ground.png", "wood.png", "teximg.jpg"}
	imgs := make([]image.Image, len(imgFiles))
	for i, fnm := range imgFiles {
		imgs[i], _, err = imagex.OpenFS(images.Images, fnm)
		if err != nil {
			errors.Log(err)
		}
	}

	// icons loaded into a texture array
	iconFiles := []string{"sound1.png", "text.png", "up.png", "world1.png"}
	iconImgs := make([]image.Image, len(iconFiles))
	for i, fnm := range iconFiles {
		iconImgs[i], _, _ = imagex.OpenFS(images.Images, fnm)
	}

	rendImgs := func(idx int) {
		drw.StartDraw()
		drw.UseGoImage(imgs[idx])
		drw.Scale(sf.Format.Bounds(), image.ZR, gpudraw.Src, gpu.NoFlipY, 0)
		for i := range imgFiles {
			// dp := image.Point{rand.Intn(500), rand.Intn(500)}
			dp := image.Point{i * 50, i * 50}
			drw.UseGoImage(imgs[i])
			drw.Copy(dp, image.ZR, gpudraw.Src, gpu.NoFlipY)
		}
		for i := range iconFiles {
			dp := image.Point{rand.Intn(500), rand.Intn(500)}
			drw.UseGoImage(iconImgs[i])
			drw.Copy(dp, image.ZR, gpudraw.Over, gpu.NoFlipY)
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
		drw.StartDraw()
		for i := 0; i < 5; i++ {
			sp := image.Point{rand.Intn(500), rand.Intn(500)}
			sz := image.Point{rand.Intn(500), rand.Intn(500)}
			drw.FillRect(colors[i%nclr], image.Rectangle{Min: sp, Max: sp.Add(sz)}, draw.Src)
		}
		drw.EndDraw()
	}
	_ = fillRnd

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
			if !pollEvents() {
				exitC <- struct{}{}
				continue
			}
			renderFrame()
		}
	}
}
