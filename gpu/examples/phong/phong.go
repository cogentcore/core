// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"runtime"
	"time"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/phong"
	"cogentcore.org/core/gpu/shape"
	"cogentcore.org/core/math32"
)

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	gp := gpu.NewGPU()
	gpu.Debug = true
	gp.Config("drawidx")

	width, height := 1024, 768
	sp, terminate, pollEvents, err := gpu.GLFWCreateWindow(gp, width, height, "Draw Triangle Indexed")
	if err != nil {
		return
	}

	sf := gpu.NewSurface(gp, sp, width, height)
	ph := phong.NewPhong(sf.GPU, &sf.Device, &sf.Format)

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		ph.Release()
		sf.Release()
		gp.Release()
		terminate()
	}

	/////////////////////////////
	// Lights

	ph.AddAmbientLight(math32.NewVector3Color(color.White).MulScalar(.1))
	ph.AddDirLight(math32.NewVector3Color(color.White), math32.Vec3(0, 1, 1))

	// ph.AddPointLight(math32.NewVector3Color(color.White), math32.Vec3(0, 2, 5), .1, .01)
	// ph.AddSpotLight(math32.NewVector3Color(color.White), math32.Vec3(-2, 5, -2), math32.Vec3(0, -1, 0), 10, 45, .01, .001)

	/////////////////////////////
	// Meshes

	// Note: 100 segs improves lighting differentiation significantly

	ph.AddMeshFromShape("floor",
		shape.NewPlane(math32.Y, 100, 100).SetSegs(math32.Vector2{100, 100}))
	ph.AddMeshFromShape("cube",
		shape.NewBox(1, 1, 1).SetSegs(math32.Vector3{100, 100, 100}))
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
	imgs := make([]image.Texture, len(imgFiles))
	for i, fnm := range imgFiles {
		pnm := filepath.Join("../images", fnm)
		imgs[i] = OpenTexture(pnm)
		ph.AddTexture(fnm, phong.NewTexture(imgs[i]))
	}

	ph.Config()

	/////////////////////////////
	// Colors

	dark := color.RGBA{20, 20, 20, 255}
	blue := color.RGBA{0, 0, 255, 255}
	blueTr := color.RGBA{0, 0, 200, 200}
	red := color.RGBA{255, 0, 0, 255}
	redTr := color.RGBA{200, 0, 0, 200}
	green := color.RGBA{0, 255, 0, 255}
	orange := color.RGBA{180, 130, 0, 255}
	tan := color.RGBA{210, 180, 140, 255}
	ph.AddColor("blue", phong.NewColors(blue, color.Black, 30, 1, 1))
	ph.AddColor("blueTr", phong.NewColors(blueTr, color.Black, 30, 1, 1))
	ph.AddColor("red", phong.NewColors(red, color.Black, 30, 1, 1))
	ph.AddColor("redTr", phong.NewColors(redTr, color.Black, 30, 1, 1))
	ph.AddColor("green", phong.NewColors(dark, green, 30, .1, 1))
	ph.AddColor("orange", phong.NewColors(orange, color.Black, 30, 1, 1))
	ph.AddColor("tan", phong.NewColors(tan, color.Black, 30, 1, 1))

	/////////////////////////////
	// Camera / Matrix

	// This is the standard camera view projection computation
	campos := math32.Vec3(0, 2, 10)
	view := phong.CameraViewMat(campos, math32.Vec3(0, 0, 0), math32.Vec3(0, 1, 0))

	aspect := sf.Format.Aspect()
	var projection math32.Matrix4
	projection.SetVkPerspective(45, aspect, 0.01, 100)

	var model1 math32.Matrix4
	model1.SetRotationY(0.5)

	var model2 math32.Matrix4
	model2.SetTranslation(-2, 0, 0)

	var model3 math32.Matrix4
	model3.SetTranslation(0, 0, -2)

	var model4 math32.Matrix4
	model4.SetTranslation(-1, 0, -2)

	var model5 math32.Matrix4
	model5.SetTranslation(1, 0, -1)

	var floortx math32.Matrix4
	floortx.SetTranslation(0, -2, -2)

	/////////////////////////////
	//  Config!

	ph.Config()

	ph.SetViewProjection(view, &projection)

	/////////////////////////////
	//  Set Mesh values

	vertexArray, normArray, textureArray, _, indexArray := ph.MeshFloatsByName("floor")
	floor.Set(vertexArray, normArray, textureArray, indexArray)
	ph.ModMeshByName("floor")

	vertexArray, normArray, textureArray, _, indexArray = ph.MeshFloatsByName("cube")
	cube.Set(vertexArray, normArray, textureArray, indexArray)
	ph.ModMeshByName("cube")

	vertexArray, normArray, textureArray, _, indexArray = ph.MeshFloatsByName("sphere")
	sphere.Set(vertexArray, normArray, textureArray, indexArray)
	ph.ModMeshByName("sphere")

	vertexArray, normArray, textureArray, _, indexArray = ph.MeshFloatsByName("cylinder")
	cylinder.Set(vertexArray, normArray, textureArray, indexArray)
	ph.ModMeshByName("cylinder")

	vertexArray, normArray, textureArray, _, indexArray = ph.MeshFloatsByName("cone")
	cone.Set(vertexArray, normArray, textureArray, indexArray)
	ph.ModMeshByName("cone")

	vertexArray, normArray, textureArray, _, indexArray = ph.MeshFloatsByName("capsule")
	capsule.Set(vertexArray, normArray, textureArray, indexArray)
	ph.ModMeshByName("capsule")

	vertexArray, normArray, textureArray, _, indexArray = ph.MeshFloatsByName("torus")
	torus.Set(vertexArray, normArray, textureArray, indexArray)
	ph.ModMeshByName("torus")

	vertexArray, normArray, textureArray, _, indexArray = ph.MeshFloatsByName("lines")
	lines.Set(vertexArray, normArray, textureArray, indexArray)
	ph.ModMeshByName("lines")

	ph.Sync()

	updateMats := func() {
		aspect := sf.Format.Aspect()
		view = phong.CameraViewMat(campos, math32.Vec3(0, 0, 0), math32.Vec3(0, 1, 0))
		projection.SetVkPerspective(45, aspect, 0.01, 100)
		ph.SetViewProjection(view, &projection)
	}

	render1 := func() {
		ph.UseColorName("blue")
		ph.SetModelMatrix(&floortx)
		ph.UseMeshName("floor")
		// ph.UseNoTexture()
		ph.UseTexturePars(math32.Vec2(50, 50), math32.Vector2{})
		ph.UseTextureName("ground.png")
		ph.Render()

		ph.UseColorName("red")
		ph.SetModelMatrix(&model2)
		ph.UseMeshName("cube")
		ph.UseFullTexture()
		ph.UseTextureName("teximg.jpg")
		// ph.UseNoTexture()
		ph.Render()

		ph.UseColorName("blue")
		ph.SetModelMatrix(&model3)
		ph.UseMeshName("cylinder")
		ph.UseTextureName("wood.png")
		// ph.UseNoTexture()
		ph.Render()

		ph.UseColorName("green")
		ph.SetModelMatrix(&model4)
		ph.UseMeshName("cone")
		// ph.UseTextureName("teximg.jpg")
		ph.UseNoTexture()
		ph.Render()

		ph.UseColorName("orange")
		ph.SetModelMatrix(&model5)
		ph.UseMeshName("lines")
		ph.UseNoTexture()
		ph.Render()

		// ph.UseColorName("blueTr")
		ph.UseColorName("tan")
		ph.SetModelMatrix(&model5)
		ph.UseMeshName("capsule")
		ph.UseNoTexture()
		ph.Render()

		// trans at end

		ph.UseColorName("redTr")
		ph.SetModelMatrix(&model1)
		ph.UseMeshName("sphere")
		ph.UseNoTexture()
		ph.Render()

		ph.UseColorName("blueTr")
		ph.SetModelMatrix(&model5)
		ph.UseMeshName("torus")
		ph.UseNoTexture()
		ph.Render()

	}

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		idx, ok := sf.AcquireNextTexture()
		if !ok {
			return
		}
		cmd := sy.CmdPool.Buff
		descIndex := 0 // if running multiple frames in parallel, need diff sets
		sy.ResetBeginRenderPass(cmd, sf.Frames[idx], descIndex)

		fcr := frameCount % 10
		_ = fcr

		campos.X = float32(frameCount) * 0.01
		campos.Z = 10 - float32(frameCount)*0.03
		updateMats()
		render1()

		frameCount++

		sy.EndRenderPass(cmd)

		sf.SubmitRender(cmd) // this is where it waits for the 16 msec
		sf.PresentTexture(idx)

		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 10 {
			fps := float64(frameCount) / dur
			fmt.Printf("fps: %.0f\n", fps)
			frameCount = 0
			stTime = eTime
		}
	}

	pollEvents()
	renderFrame()
	pollEvents()

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
