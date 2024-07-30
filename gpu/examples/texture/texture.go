// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"path/filepath"
	"runtime"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
)

func init() {
	// a must lock main thread for gpu!
	runtime.LockOSThread()
}

type CamView struct {
	Model      math32.Matrix4
	View       math32.Matrix4
	Projection math32.Matrix4
}

func main() {
	gp := gpu.NewGPU()
	gpu.Debug = true
	gp.Config("texture")

	width, height := 1024, 768
	sp, terminate, pollEvents, err := gpu.GLFWCreateWindow(gp, width, height, "Draw Triangle Indexed")
	if err != nil {
		return
	}

	sf := gpu.NewSurface(gp, sp, width, height)

	fmt.Printf("format: %s\n", sf.Format.String())

	sy := gp.NewGraphicsSystem("texture", sf.Device)

	destroy := func() {
		sy.WaitDone()
		sy.Release()
		sf.Release()
		gp.Release()
		terminate()
	}

	pl := sy.AddGraphicsPipeline("texture")
	sy.ConfigRender(&sf.Format, gpu.Depth32)
	// sf.SetRender(&sy.Render)
	sy.SetClearColor(color.RGBA{50, 50, 50, 255})

	sh := pl.AddShader("texture")
	sh.OpenFile("texture.wgsl")
	pl.AddEntry(sh, gpu.VertexShader, "vs_main")
	pl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	vgp := sy.Vars.AddVertexGroup()
	tgp := sy.Vars.AddGroup(gpu.SampledTexture) // texture in 0 so frag only gets 0
	ugp := sy.Vars.AddGroup(gpu.Uniform)

	nPts := 4
	nIndexes := 6

	posv := vgp.Add("Pos", gpu.Float32Vector3, nPts, gpu.VertexShader)
	clrv := vgp.Add("Color", gpu.Float32Vector3, nPts, gpu.VertexShader)
	txcv := vgp.Add("TexCoord", gpu.Float32Vector2, nPts, gpu.VertexShader)
	idxv := vgp.Add("Index", gpu.Uint16, nIndexes, gpu.VertexShader)
	idxv.Role = gpu.Index

	camv := ugp.AddStruct("Camera", gpu.Float32Matrix4.Bytes()*3, 1, gpu.VertexShader)

	txv := tgp.Add("TexSampler", gpu.TextureRGBA32, 1, gpu.FragmentShader)

	vgp.SetNValues(1)
	ugp.SetNValues(1)
	tgp.SetNValues(3)

	imgFiles := []string{"ground.png", "wood.png", "teximg.jpg"}
	imgs := make([]image.Image, len(imgFiles))
	for i, fnm := range imgFiles {
		pnm := filepath.Join("../images", fnm)
		imgs[i], _, _ = imagex.Open(pnm)
		img := txv.Values.Values[i]
		img.SetFromGoImage(imgs[i], 0, gpu.NoFlipY)
		// img.Texture.Sampler.Border = gpu.BorderBlack
		// img.Texture.Sampler.UMode = gpu.ClampToBorder
		// img.Texture.Sampler.VMode = gpu.ClampToBorder
	}

	sy.Config()

	rectPos := posv.Values.Values[0]
	gpu.SetValueFrom(rectPos, []float32{
		-0.5, -0.5, 0.0,
		0.5, -0.5, 0.0,
		0.5, 0.5, 0.0,
		-0.5, 0.5, 0.0})

	rectClr := clrv.Values.Values[0]
	gpu.SetValueFrom(rectClr, []float32{
		1.0, 0.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0,
		1.0, 1.0, 0.0})

	rectTex := txcv.Values.Values[0]
	gpu.SetValueFrom(rectTex, []float32{
		1.0, 0.0,
		0.0, 0.0,
		0.0, 1.0,
		1.0, 1.0})

	rectIndex := idxv.Values.Values[0]
	gpu.SetValueFrom(rectIndex, []uint16{0, 1, 2, 0, 2, 3})

	// This is the standard camera view projection computation
	cam := camv.Values.Values[0]
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
	// fmt.Printf("aspect: %g\n", aspect)
	// VkPerspective version automatically flips Y axis and shifts depth
	// into a 0..1 range instead of -1..1, so original GL based geometry
	// will render identically here.
	camo.Projection.SetVkPerspective(45, aspect, 0.01, 100)

	gpu.SetValueFrom(cam, []CamView{camo}) // note: always use slice to copy

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		camo.Model.SetRotationY(.004 * float32(frameCount))
		gpu.SetValueFrom(cam, []CamView{camo})

		imgIndex := int32(frameCount % len(imgs))
		_ = imgIndex

		view, err := sf.AcquireNextTexture()
		if err != nil {
			slog.Error(err.Error())
			return
		}
		cmd := sy.NewCommandEncoder()
		rp := sy.BeginRenderPass(cmd, view)
		pl.BindPipeline(rp)
		pl.BindDrawVertex(rp)
		rp.End()
		sf.SubmitRender(cmd)
		sf.Present()

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
			if !pollEvents() {
				exitC <- struct{}{}
				continue
			}
			renderFrame()
		}
	}
}
