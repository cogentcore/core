// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"runtime"
	"time"
	"unsafe"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/gpudraw"
	"cogentcore.org/core/math32"
	"github.com/cogentcore/webgpu/wgpu"
)

//go:embed indexed.wgsl
var indexed string

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

type CamView struct {
	Model      math32.Matrix4
	View       math32.Matrix4
	Projection math32.Matrix4
}

func main() {
	if gpu.Init() != nil {
		return
	}

	gp := gpu.NewGPU()
	gpu.Debug = true
	gp.Config("drawidx")

	var resize func(size image.Point)
	size := image.Point{1024, 768}
	sp, terminate, pollEvents, size, err := gpu.GLFWCreateWindow(gp, size, "Offscreen Render", &resize)
	if err != nil {
		return
	}

	sf := gpu.NewSurface(gp, sp, size, 1, gpu.UndefinedType)
	drw := gpudraw.NewDrawer(gp, sf)

	rt := gpu.NewRenderTexture(gp, sf.Device(), size, 4, gpu.UndefinedType)
	sy := gpu.NewGraphicsSystem(gp, "drawidx", rt)
	fmt.Println("format:", rt.Frames[0].Format.String())

	resize = func(size image.Point) {
		sf.SetSize(size)
		rt.SetSize(size)
	}
	destroy := func() {
		drw.Release()
		sy.Release()
		sf.Release()
		gp.Release()
		terminate()
	}

	pl := sy.AddGraphicsPipeline("drawidx")
	pl.SetCullMode(wgpu.CullModeNone)
	sy.SetClearColor(color.RGBA{50, 50, 50, 255})

	sh := pl.AddShader("indexed")
	sh.OpenCode(indexed)
	pl.AddEntry(sh, gpu.VertexShader, "vs_main")
	pl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	vgp := sy.Vars().AddVertexGroup()
	ugp := sy.Vars().AddGroup(gpu.Uniform)

	// vertex are dynamically sized in general, so using 0 here
	posv := vgp.Add("Pos", gpu.Float32Vector3, 0, gpu.VertexShader)
	clrv := vgp.Add("Color", gpu.Float32Vector3, 0, gpu.VertexShader)
	// note: index goes last usually
	idxv := vgp.Add("Index", gpu.Uint16, 0, gpu.VertexShader)
	idxv.Role = gpu.Index

	camv := ugp.AddStruct("Camera", int(unsafe.Sizeof(CamView{})), 1, gpu.VertexShader)

	vgp.SetNValues(1)
	ugp.SetNValues(1)
	sy.Config()

	triPos := posv.Values.Values[0]
	gpu.SetValueFrom(triPos, []float32{
		-0.5, 0.5, 0.0,
		0.5, 0.5, 0.0,
		0.0, -0.5, 0.0}) // negative point is UP in native Vulkan

	triClr := clrv.Values.Values[0]
	gpu.SetValueFrom(triClr, []float32{
		1.0, 0.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0})

	triIndex := idxv.Values.Values[0]
	gpu.SetValueFrom(triIndex, []uint16{0, 1, 2})
	// note: the only way to set indexes is at start..

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
	camo.Projection.SetPerspective(45, aspect, 0.01, 100)
	gpu.SetValueFrom(cam, []CamView{camo}) // note: always use slice to copy

	updateAspect := func() {
		aspect := rt.Format.Aspect()
		fmt.Printf("aspect: %g\n", aspect)
		camo.Projection.SetPerspective(45, aspect, 0.01, 100)
		gpu.SetValueFrom(cam, []CamView{camo}) // note: always use slice to copy
	}

	updateAspect()

	// todo:
	// drw.ConfigTexture(0, &rt.Format)

	frameCount := 0
	stTime := time.Now()

	didSave := false
	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		camo.Model.SetRotationY(.1 * float32(frameCount))
		gpu.SetValueFrom(cam, []CamView{camo})

		rt.Frames[0].ConfigReadBuffer() // ensure configed with proper size

		rp, err := sy.BeginRenderPass()
		if err != nil {
			return
		}
		pl.BindPipeline(rp)
		pl.BindDrawIndexed(rp)
		rp.End()
		if !didSave {
			didSave = true
			rt.Frames[0].CopyToReadBuffer(sy.CommandEncoder)
			sy.EndRenderPass(rp)
			img, err := rt.Frames[0].ReadGoImage()
			if errors.Log(err) != nil {
				return
			}
			imagex.Save(img, "render.png")
			fmt.Println("saved image", rt.Frames[0].ReadBufferDims)
		} else {
			sy.EndRenderPass(rp)
		}

		// todo:
		// drw.SetFrameTexture(0, fr)
		// drw.SyncTextures()
		// drw.StartDraw(0)
		// drw.Scale(0, 0, sf.Format.Bounds(), image.ZR, draw.Src, gpu.NoFlipY, 0)
		// drw.EndDraw()

		// fmt.Printf("present %v\n\n", time.Now().Sub(rt))
		frameCount++
		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 10 {
			fps := float64(frameCount) / dur
			fmt.Printf("fps: %.0f\n", fps)
			sz := rt.Format.Size
			sz.X -= 10
			rt.SetSize(sz)
			updateAspect()
			frameCount = 0
			stTime = eTime
		}
	}

	exitC := make(chan struct{}, 2)

	fpsDelay := time.Second / 6
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
