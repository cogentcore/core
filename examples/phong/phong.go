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
	"path/filepath"
	"runtime"
	"time"

	"github.com/goki/mat32"
	vk "github.com/goki/vulkan"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vgpu/vgpu"
	"github.com/goki/vgpu/vphong"
	"github.com/goki/vgpu/vshape"
)

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

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
	window, err := glfw.CreateWindow(1280, 960, "vPhong Test", nil, nil)
	vgpu.IfPanic(err)

	// note: for graphics, require these instance extensions before init gpu!
	winext := window.GetRequiredInstanceExtensions()
	gp := vgpu.NewGPU()
	gp.AddInstanceExt(winext...)
	//	vgpu.Debug = true
	gp.Config("vPhong test")

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
	sy.ConfigRender(&sf.Format, vgpu.Depth32)
	sf.SetRender(&sy.Render)
	ph.ConfigSys()
	sy.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		ph.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		vgpu.Terminate()
	}

	/////////////////////////////
	// Lights

	amblt := mat32.NewVec3Color(color.White).MulScalar(.1)
	ph.AddAmbientLight(amblt)

	dirlt := mat32.NewVec3Color(color.White).MulScalar(1)
	ph.AddDirLight(dirlt, mat32.Vec3{0, 1, 1})

	// ph.AddPointLight(mat32.NewVec3Color(color.White), mat32.Vec3{0, 2, 5}, .1, .01)
	//
	// ph.AddSpotLight(mat32.NewVec3Color(color.White), mat32.Vec3{-2, 5, -2}, mat32.Vec3{0, -1, 0}, 10, 45, .01, .001)

	/////////////////////////////
	// Meshes

	floor := vshape.NewPlane(mat32.Y, 100, 100)
	floor.Segs.Set(100, 100) // won't show lighting without
	nVtx, nIdx := floor.N()
	ph.AddMesh("floor", nVtx, nIdx, false)

	cube := vshape.NewBox(1, 1, 1)
	cube.Segs.Set(100, 100, 100) // key for showing lights
	nVtx, nIdx = cube.N()
	ph.AddMesh("cube", nVtx, nIdx, false)

	sphere := vshape.NewSphere(.5, 64)
	nVtx, nIdx = sphere.N()
	ph.AddMesh("sphere", nVtx, nIdx, false)

	cylinder := vshape.NewCylinder(1, .5, 64, 64, true, true)
	nVtx, nIdx = cylinder.N()
	ph.AddMesh("cylinder", nVtx, nIdx, false)

	cone := vshape.NewCone(1, .5, 64, 64, true)
	nVtx, nIdx = cone.N()
	ph.AddMesh("cone", nVtx, nIdx, false)

	capsule := vshape.NewCapsule(1, .5, 64, 64)
	// capsule.BotRad = 0
	nVtx, nIdx = capsule.N()
	ph.AddMesh("capsule", nVtx, nIdx, false)

	torus := vshape.NewTorus(2, .2, 64)
	nVtx, nIdx = torus.N()
	ph.AddMesh("torus", nVtx, nIdx, false)

	lines := vshape.NewLines([]mat32.Vec3{{-3, -1, 0}, {-2, 1, 0}, {2, 1, 0}, {3, -1, 0}}, mat32.Vec2{.2, .1}, false)
	nVtx, nIdx = lines.N()
	ph.AddMesh("lines", nVtx, nIdx, false)

	/////////////////////////////
	// Textures

	imgFiles := []string{"ground.png", "wood.png", "teximg.jpg"}
	imgs := make([]image.Image, len(imgFiles))
	for i, fnm := range imgFiles {
		pnm := filepath.Join("../images", fnm)
		imgs[i] = OpenImage(pnm)
		ph.AddTexture(fnm, vphong.NewTexture(imgs[i]))
	}

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
	ph.AddColor("blue", vphong.NewColors(blue, color.Black, 30, 1, 1))
	ph.AddColor("blueTr", vphong.NewColors(blueTr, color.Black, 30, 1, 1))
	ph.AddColor("red", vphong.NewColors(red, color.Black, 30, 1, 1))
	ph.AddColor("redTr", vphong.NewColors(redTr, color.Black, 30, 1, 1))
	ph.AddColor("green", vphong.NewColors(dark, green, 30, .1, 1))
	ph.AddColor("orange", vphong.NewColors(orange, color.Black, 30, 1, 1))
	ph.AddColor("tan", vphong.NewColors(tan, color.Black, 30, 1, 1))

	/////////////////////////////
	// Camera / Mtxs

	// This is the standard camera view projection computation
	campos := mat32.Vec3{0, 2, 10}
	view := vphong.CameraViewMat(campos, mat32.Vec3{0, 0, 0}, mat32.Vec3Y)

	aspect := sf.Format.Aspect()
	var prjn mat32.Mat4
	prjn.SetVkPerspective(45, aspect, 0.01, 100)

	var model1 mat32.Mat4
	model1.SetRotationY(0.5)

	var model2 mat32.Mat4
	model2.SetTranslation(-2, 0, 0)

	var model3 mat32.Mat4
	model3.SetTranslation(0, 0, -2)

	var model4 mat32.Mat4
	model4.SetTranslation(-1, 0, -2)

	var model5 mat32.Mat4
	model5.SetTranslation(1, 0, -1)

	var floortx mat32.Mat4
	floortx.SetTranslation(0, -2, -2)

	/////////////////////////////
	//  Config!

	ph.Config()

	ph.SetViewPrjn(view, &prjn)

	/////////////////////////////
	//  Set Mesh values

	vtxAry, normAry, texAry, _, idxAry := ph.MeshFloatsByName("floor")
	floor.Set(vtxAry, normAry, texAry, idxAry)
	ph.ModMeshByName("floor")

	vtxAry, normAry, texAry, _, idxAry = ph.MeshFloatsByName("cube")
	cube.Set(vtxAry, normAry, texAry, idxAry)
	ph.ModMeshByName("cube")

	vtxAry, normAry, texAry, _, idxAry = ph.MeshFloatsByName("sphere")
	sphere.Set(vtxAry, normAry, texAry, idxAry)
	ph.ModMeshByName("sphere")

	vtxAry, normAry, texAry, _, idxAry = ph.MeshFloatsByName("cylinder")
	cylinder.Set(vtxAry, normAry, texAry, idxAry)
	ph.ModMeshByName("cylinder")

	vtxAry, normAry, texAry, _, idxAry = ph.MeshFloatsByName("cone")
	cone.Set(vtxAry, normAry, texAry, idxAry)
	ph.ModMeshByName("cone")

	vtxAry, normAry, texAry, _, idxAry = ph.MeshFloatsByName("capsule")
	capsule.Set(vtxAry, normAry, texAry, idxAry)
	ph.ModMeshByName("capsule")

	vtxAry, normAry, texAry, _, idxAry = ph.MeshFloatsByName("torus")
	torus.Set(vtxAry, normAry, texAry, idxAry)
	ph.ModMeshByName("torus")

	vtxAry, normAry, texAry, _, idxAry = ph.MeshFloatsByName("lines")
	lines.Set(vtxAry, normAry, texAry, idxAry)
	ph.ModMeshByName("lines")

	ph.Sync()

	updateMats := func() {
		aspect := sf.Format.Aspect()
		view = vphong.CameraViewMat(campos, mat32.Vec3{0, 0, 0}, mat32.Vec3Y)
		prjn.SetVkPerspective(45, aspect, 0.01, 100)
		ph.SetViewPrjn(view, &prjn)
	}

	render1 := func() {
		ph.UseColorName("blue")
		ph.SetModelMtx(&floortx)
		ph.UseMeshName("floor")
		// ph.UseNoTexture()
		ph.UseTexturePars(mat32.Vec2{50, 50}, mat32.Vec2{})
		ph.UseTextureName("ground.png")
		ph.Render()

		ph.UseColorName("red")
		ph.SetModelMtx(&model2)
		ph.UseMeshName("cube")
		ph.UseFullTexture()
		ph.UseTextureName("teximg.jpg")
		// ph.UseNoTexture()
		ph.Render()

		ph.UseColorName("blue")
		ph.SetModelMtx(&model3)
		ph.UseMeshName("cylinder")
		ph.UseTextureName("wood.png")
		// ph.UseNoTexture()
		ph.Render()

		ph.UseColorName("green")
		ph.SetModelMtx(&model4)
		ph.UseMeshName("cone")
		// ph.UseTextureName("teximg.jpg")
		ph.UseNoTexture()
		ph.Render()

		ph.UseColorName("orange")
		ph.SetModelMtx(&model5)
		ph.UseMeshName("lines")
		ph.UseNoTexture()
		ph.Render()

		// ph.UseColorName("blueTr")
		ph.UseColorName("tan")
		ph.SetModelMtx(&model5)
		ph.UseMeshName("capsule")
		ph.UseNoTexture()
		ph.Render()

		// trans at end

		ph.UseColorName("redTr")
		ph.SetModelMtx(&model1)
		ph.UseMeshName("sphere")
		ph.UseNoTexture()
		ph.Render()

		ph.UseColorName("blueTr")
		ph.SetModelMtx(&model5)
		ph.UseMeshName("torus")
		ph.UseNoTexture()
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

		campos.X = float32(frameCount) * 0.01
		campos.Z = 10 - float32(frameCount)*0.03
		updateMats()
		render1()

		frameCount++

		sy.EndRenderPass(cmd)

		sf.SubmitRender(cmd) // this is where it waits for the 16 msec
		sf.PresentImage(idx)

		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 10 {
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

	fpsDelay := time.Second / 60
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
