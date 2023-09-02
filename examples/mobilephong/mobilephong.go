package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"path/filepath"
	"time"

	"github.com/goki/mat32"
	vk "github.com/goki/vulkan"
	"goki.dev/mobile/app"
	"goki.dev/mobile/event/lifecycle"
	"goki.dev/mobile/event/paint"
	"goki.dev/mobile/event/size"
	"goki.dev/mobile/event/touch"
	"goki.dev/vgpu/v2/vgpu"
	"goki.dev/vgpu/v2/vphong"
	"goki.dev/vgpu/v2/vshape"
)

//go:embed images/*.png
//go:embed images/*.jpg
var content embed.FS

var (
	gpu      *vgpu.GPU
	system   *vgpu.System
	surface  *vgpu.Surface
	pipeline *vgpu.Pipeline
	ph       *vphong.Phong
	window   uintptr

	campos mat32.Vec3
	view   *mat32.Mat4
	prjn   mat32.Mat4

	model1  mat32.Mat4
	model2  mat32.Mat4
	model3  mat32.Mat4
	model4  mat32.Mat4
	model5  mat32.Mat4
	floortx mat32.Mat4

	frameCount int
	stTime     time.Time
	fpsDelay   time.Duration
	fpsTicker  *time.Ticker

	touchX float32
	touchY float32
)

func OpenImage(fname string) image.Image {
	file, err := content.Open(fname)
	if err != nil {
		log.Printf("image: %s\n", err)
		return nil
	}
	defer file.Close()
	gimg, _, err := image.Decode(file)
	if err != nil {
		log.Println(err)
	}
	return gimg
}

func onStart(a app.App) {
	winext := vk.GetRequiredInstanceExtensions()
	log.Printf("required exts: %#v\n", winext)
	gpu = vgpu.NewGPU()
	gpu.AddInstanceExt(winext...)
	gpu.Config("phong")

	var sf vk.Surface
	log.Println("in onStart", gpu.Instance, window, &sf)
	ret := vk.CreateWindowSurface(gpu.Instance, window, nil, &sf)
	if err := vk.Error(ret); err != nil {
		log.Println("vulkan error:", err)
		return
	}
	surface = vgpu.NewSurface(gpu, sf)

	log.Printf("format: %s\n", surface.Format.String())

	ph = &vphong.Phong{}
	system = &ph.Sys
	system.InitGraphics(gpu, "vphong.Phong", &surface.Device)
	system.ConfigRender(&surface.Format, vgpu.Depth32)
	surface.SetRender(&system.Render)
	ph.ConfigSys()
	system.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)

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
		pnm := filepath.Join("images", fnm)
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
	campos = mat32.Vec3{0, 2, 10}
	view = vphong.CameraViewMat(campos, mat32.Vec3{0, 0, 0}, mat32.Vec3Y)

	aspect := surface.Format.Aspect()
	prjn.SetVkPerspective(45, aspect, 0.01, 100)

	model1.SetRotationY(0.5)

	model2.SetTranslation(-2, 0, 0)

	model3.SetTranslation(0, 0, -2)

	model4.SetTranslation(-1, 0, -2)

	model5.SetTranslation(1, 0, -1)

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

	// system = gpu.NewGraphicsSystem("phong", &surface.Device)
	// pipeline = system.NewPipeline("phong")
	// system.ConfigRender(&surface.Format, vgpu.UndefType)
	// surface.SetRender(&system.Render)

	// pipeline.AddShaderEmbed("trianglelit", vgpu.VertexShader, content, "trianglelit.spv")
	// pipeline.AddShaderEmbed("vtxcolor", vgpu.FragmentShader, content, "vtxcolor.spv")

	// system.Config()

	frameCount = 0
	stTime = time.Now()
	fpsDelay = time.Second / 60
	fpsTicker = time.NewTicker(fpsDelay)

	go func() {
		for {
			select {
			case <-fpsTicker.C:
				if system == nil {
					log.Println("stopped because system is nil")
					return
				}
				a.Send(paint.Event{})
				a.Publish()
			}
		}
	}()
}

func onStop() {
	vk.DeviceWaitIdle(surface.Device.Device)
	ph.Destroy()
	system.Destroy()
	system = nil
	surface.Destroy()
	gpu.Destroy()
	vgpu.Terminate()
}

func updateMats() {
	aspect := surface.Format.Aspect()
	view = vphong.CameraViewMat(campos, mat32.Vec3{0, 0, 0}, mat32.Vec3Y)
	prjn.SetVkPerspective(45, aspect, 0.01, 100)
	ph.SetViewPrjn(view, &prjn)
}

func render1() {
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

func onPaint() {
	if system != nil {
		idx := surface.AcquireNextImage()
		cmd := system.CmdPool.Buff
		descIdx := 0 // if running multiple frames in parallel, need diff sets
		system.ResetBeginRenderPass(cmd, surface.Frames[idx], descIdx)

		fcr := frameCount % 10
		_ = fcr

		campos.X = float32(frameCount) * 0.01
		campos.Z = 10 - float32(frameCount)*0.03
		updateMats()
		render1()

		frameCount++

		system.EndRenderPass(cmd)

		surface.SubmitRender(cmd) // this is where it waits for the 16 msec
		surface.PresentImage(idx)

		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 10 {
			fps := float64(frameCount) / dur
			fmt.Printf("fps: %.0f\n", fps)
			frameCount = 0
			stTime = eTime
		}
		// log.Println("painted")

		// https://source.android.com/devices/graphics/arch-gameloops
		// FPS may drop down when no interacton with the app, should skip frames there.
		// TODO: use VK_GOOGLE_display_timing_enabled as cool guys would do. Don't be an uncool fool.
		// if lastRender > fpsDelay {
		// 	// skip frame
		// 	lastRender = lastRender - fpsDelay
		// 	continue
		// }
		// ts := time.Now()
	}
}

func main() {
	app.Main(func(a app.App) {
		log.SetPrefix("GoMobileVulkan: ")
		vgpu.Debug = true
		orPanic(vk.SetDefaultGetInstanceProcAddr())
		orPanic(vk.Init())

		var sz size.Event
		for e := range a.Events() {
			switch e := a.Filter(e).(type) {
			case lifecycle.Event:
				switch e.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					window = a.Window()
					log.Println("on start, window uintptr:", window)
					onStart(a)
				case lifecycle.CrossOff:
					log.Println("on stop")
					onStop()
				}
			case size.Event:
				log.Println("size event")
				sz = e
				touchX = float32(sz.WidthPx / 2)
				touchY = float32(sz.HeightPx / 2)
			case paint.Event:
				// log.Println("paint event")
				onPaint()
			case touch.Event:
				log.Println("touch event", e)
				touchX = e.X
				touchY = e.Y
			}
		}
	})
}

func orPanic(err interface{}) {
	switch v := err.(type) {
	case error:
		if v != nil {
			panic(err)
		}
	case vk.Result:
		if err := vk.Error(v); err != nil {
			panic(err)
		}
	case bool:
		if !v {
			panic("condition failed: != true")
		}
	}
}
