// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"image/color"
	"runtime"
	"strconv"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/examples/images"
	"cogentcore.org/core/gpu/phong"
	"cogentcore.org/core/gpu/shape"
	"cogentcore.org/core/math32"
	"github.com/cogentcore/webgpu/wgpu"
)

func init() {
	// must lock main thread for gpu!  really
	runtime.LockOSThread()
}

type Object struct {
	Mesh    string
	Color   color.Color
	Texture string
	Matrix  math32.Matrix4
}

func main() {
	gp := gpu.NewGPU()
	gpu.Debug = true
	gp.Config("phong")

	var resize func(size image.Point)
	size := image.Point{1024, 768}
	sp, terminate, pollEvents, size, err := gpu.GLFWCreateWindow(gp, size, "Phong", &resize)
	if err != nil {
		return
	}

	sf := gpu.NewSurface(gp, sp, size, 1, gpu.Depth32)
	ph := phong.NewPhong(sf.GPU, sf)
	fmt.Printf("format: %s\n", sf.Format.String())
	resize = func(size image.Point) { sf.SetSize(size) }
	destroy := func() {
		ph.Release()
		sf.Release()
		gp.Release()
		terminate()
	}

	/////////////////////////////
	// Lights

	ph.AddAmbient(math32.NewVector3Color(color.White).MulScalar(.1))
	ph.AddDirectional(math32.NewVector3Color(color.White), math32.Vec3(0, 1, 1))

	// ph.AddPoint(math32.NewVector3Color(color.White), math32.Vec3(0, 2, 5), .1, .01)
	// ph.AddSpot(math32.NewVector3Color(color.White), math32.Vec3(-2, 5, -2), math32.Vec3(0, -1, 0), 10, 45, .01, .001)

	/////////////////////////////
	// Meshes

	// Note: 100 segs improves lighting differentiation significantly

	ph.AddMeshFromShape("floor",
		shape.NewPlane(math32.Y, 100, 100).SetSegs(math32.Vector2i{100, 100}).SetNormalNeg(true))
	ph.AddMeshFromShape("cube",
		shape.NewBox(1, 1, 1).SetSegs(math32.Vector3i{50, 50, 50}))
	ph.AddMeshFromShape("sphere", shape.NewSphere(.5, 64))
	ph.AddMeshFromShape("cylinder", shape.NewCylinder(1, .5, 64, 64, true, true))
	ph.AddMeshFromShape("cone", shape.NewCone(1, .5, 64, 64, true))
	ph.AddMeshFromShape("capsule", shape.NewCapsule(1, .5, 64, 64))
	ph.AddMeshFromShape("torus", shape.NewTorus(2, .2, 64))

	lines := shape.NewLines([]math32.Vector3{{-3, -1, 0}, {-2, 1, 0}, {2, 1, 0}, {3, -1, 0}}, math32.Vec2(.2, .1), false)
	ph.AddMeshFromShape("lines", lines)

	/////////////////////////////
	// Textures

	imgFiles := []string{"ground.png", "wood.png", "teximg.jpg"}
	imgs := make([]image.Image, len(imgFiles))
	for i, fnm := range imgFiles {
		imgs[i], _, _ = imagex.OpenFS(images.Images, fnm)
		fn := strings.Split(fnm, ".")[0]
		ph.AddTexture(fn, phong.NewTexture(imgs[i]))
	}

	/////////////////////////////
	// Colors

	dark := color.RGBA{20, 20, 20, 255}
	_ = dark
	blue := color.RGBA{0, 0, 255, 255}
	blueTr := color.RGBA{0, 0, 200, 200}
	red := color.RGBA{255, 0, 0, 255}
	redTr := color.RGBA{200, 0, 0, 200}
	green := color.RGBA{0, 255, 0, 255}
	orange := color.RGBA{180, 130, 0, 255}
	tan := color.RGBA{210, 180, 140, 255}

	/////////////////////////////
	// Camera / Matrix

	// This is the standard camera view projection computation
	campos := math32.Vec3(0, 2, 10)
	view := phong.CameraViewMat(campos, math32.Vec3(0, 0, 0), math32.Vec3(0, 1, 0))

	aspect := sf.Format.Aspect()
	var projection math32.Matrix4
	projection.SetPerspective(45, aspect, 0.01, 100)

	objs := []Object{
		{Mesh: "floor", Color: blue, Texture: "ground"},
		{Mesh: "cube", Color: red, Texture: "teximg"},
		{Mesh: "cylinder", Color: blue, Texture: "wood"},
		{Mesh: "cone", Color: green},
		{Mesh: "lines", Color: orange},
		{Mesh: "capsule", Color: tan},
		{Mesh: "sphere", Color: redTr},
		{Mesh: "torus", Color: blueTr},
	}

	objs[0].Matrix.SetTranslation(0, -2, -2)
	// objs[0].Colors.SetTextureRepeat(math32.Vector2{50, 50})
	objs[1].Matrix.SetTranslation(-2, 0, 0)
	objs[2].Matrix.SetTranslation(0, 0, -2)
	objs[3].Matrix.SetTranslation(-1, 0, -2)
	objs[4].Matrix.SetTranslation(1, 0, -1)
	objs[5].Matrix.SetTranslation(1, 0, -1)
	objs[6].Matrix.SetRotationY(0.5)
	objs[7].Matrix.SetTranslation(1, 0, -1)

	for i, ob := range objs {
		nm := strconv.Itoa(i)
		ph.AddObject(nm, phong.NewObject(&ob.Matrix, ob.Color, color.Black, 30, 1, 1))
	}

	ph.SetCamera(view, &projection)

	/////////////////////////////
	//  Config!

	ph.Config()

	updateCamera := func() {
		aspect := sf.Format.Aspect()
		view = phong.CameraViewMat(campos, math32.Vec3(0, 0, 0), math32.Vec3(0, 1, 0))
		projection.SetPerspective(45, aspect, 0.01, 100)
		ph.SetCamera(view, &projection)
	}
	updateCamera()

	updateObs := func() {
		for i, ob := range objs {
			nm := strconv.Itoa(i)
			od := ph.SetObjectMatrix(nm, &ob.Matrix)
			if i == 0 {
				od.SetTextureRepeat(math32.Vector2{50, 50})
			}
			if i == 3 { // cone
				od.SetColors(dark, green, 30, .1, 1) // emissive, not reflective
			}
		}
	}
	updateObs() // gotta do at least once

	render1 := func(rp *wgpu.RenderPassEncoder) {
		for i, ob := range objs {
			ph.UseObjectIndex(i)
			ph.UseMeshName(ob.Mesh)
			if ob.Texture != "" {
				ph.UseTextureName(ob.Texture)
			} else {
				ph.UseNoTexture()
			}
			ph.Render(rp)
		}
	}

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		fcr := frameCount % 10
		_ = fcr
		campos.X = float32(frameCount) * 0.01
		campos.Z = 10 - float32(frameCount)*0.03
		frameCount++
		updateCamera()

		rp, err := ph.RenderStart()
		if errors.Log(err) != nil {
			return
		}
		render1(rp)
		ph.RenderEnd(rp)

		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 10 {
			fps := float64(frameCount) / dur
			fmt.Printf("fps: %.0f\n", fps)
			frameCount = 0
			stTime = eTime
			sf.ReConfigSwapChain() // in case we resized..
			fmt.Println(sf.Format)
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
