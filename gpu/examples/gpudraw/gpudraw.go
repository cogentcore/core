// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"runtime"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/examples/images"
	"cogentcore.org/core/gpu/gpudraw"
	"golang.org/x/exp/rand"
)

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	gp := gpu.NewGPU()
	gp.Config("gpudraw")

	var resize func(size image.Point)
	size := image.Point{1024, 768}
	sp, terminate, pollEvents, size, err := gpu.GLFWCreateWindow(gp, size, "GPU Draw", &resize)
	if err != nil {
		return
	}

	sf := gpu.NewSurface(gp, sp, size, 1, gpu.UndefinedType) // note: 1, no depth for draw
	drw := gpudraw.NewDrawer(gp, sf)
	fmt.Printf("format: %s\n", sf.Format.String())
	resize = func(size image.Point) { sf.SetSize(size) }

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
		drw.Start()
		drw.Scale(image.Rectangle{}, imgs[idx], image.Rectangle{}, 0, gpudraw.Src, gpudraw.Unchanged)
		for i := range imgFiles {
			// dp := image.Point{rand.Intn(500), rand.Intn(500)}
			dp := image.Point{i * 50, i * 50}
			drw.Copy(dp, imgs[i], image.Rectangle{}, gpudraw.Src, gpudraw.Unchanged)
		}
		for i := range iconFiles {
			dp := image.Point{rand.Intn(500), rand.Intn(500)}
			drw.Copy(dp, iconImgs[i], image.Rectangle{}, gpudraw.Over, gpudraw.Unchanged)
		}
		drw.End()
	}

	_ = rendImgs

	red := color.RGBA{128, 0, 0, 128}
	green := color.RGBA{0, 128, 0, 128}
	blue := color.RGBA{0, 0, 255, 255}

	colors := []color.Color{color.White, color.Black, red, green, blue}

	rendFill := func() {
		nclr := len(colors)
		drw.Start()
		for i := 0; i < 5; i++ {
			// sp := image.Point{rand.Intn(500), rand.Intn(500)}
			// sz := image.Point{rand.Intn(500), rand.Intn(500)}
			sp := image.Point{i * 30, i * 40}
			sz := image.Point{(i + 1) * 20, (i + 1) * 20}
			sr := image.Rectangle{Min: sp, Max: sp.Add(sz)}
			drw.Fill(colors[i%nclr], sr, draw.Over)
		}
		drw.End()
	}
	_ = rendFill

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		fcr := frameCount % 4
		_ = fcr
		switch {
		case fcr < 3:
			rendImgs(fcr)
		default:
			rendFill()
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

	fpsDelay := time.Second / 60
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
