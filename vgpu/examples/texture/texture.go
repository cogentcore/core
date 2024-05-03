// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"

	vk "github.com/goki/vulkan"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/vgpu"
	"github.com/go-gl/glfw/v3.3/glfw"
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
	window, err := glfw.CreateWindow(1024, 768, "Draw Texture", nil, nil)
	vgpu.IfPanic(err)

	// note: for graphics, require these instance extensions before init gpu!
	winext := window.GetRequiredInstanceExtensions()
	gp := vgpu.NewGPU()
	gp.AddInstanceExt(winext...)
	vgpu.Debug = true
	gp.Config("texture")

	// gp.PropertiesString(true) // print

	surfPtr, err := window.CreateWindowSurface(gp.Instance, nil)
	if err != nil {
		log.Println(err)
		return
	}
	sf := vgpu.NewSurface(gp, vk.SurfaceFromPointer(surfPtr))

	fmt.Printf("format: %s\n", sf.Format.String())

	sy := gp.NewGraphicsSystem("texture", &sf.Device)

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		sy.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		vgpu.Terminate()
	}

	pl := sy.NewPipeline("texture")
	sy.ConfigRender(&sf.Format, vgpu.Depth32)
	sf.SetRender(&sy.Render)
	sy.SetClearColor(0.2, 0.2, 0.2, 1)
	sy.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)

	pl.AddShaderFile("texture_vert", vgpu.VertexShader, "texture_vert.spv")
	pl.AddShaderFile("texture_frag", vgpu.FragmentShader, "texture_frag.spv")

	vars := sy.Vars()
	vset := vars.AddVertexSet()
	pcset := vars.AddPushSet()
	uset := vars.AddSet()
	txset := vars.AddSet()

	nPts := 4
	nIndexes := 6

	posv := vset.Add("Pos", vgpu.Float32Vector3, nPts, vgpu.Vertex, vgpu.VertexShader)
	clrv := vset.Add("Color", vgpu.Float32Vector3, nPts, vgpu.Vertex, vgpu.VertexShader)
	txcv := vset.Add("TexCoord", vgpu.Float32Vector2, nPts, vgpu.Vertex, vgpu.VertexShader)
	// note: always put indexes last so there isn't a gap in the location indexes!
	idxv := vset.Add("Index", vgpu.Uint16, nIndexes, vgpu.Index, vgpu.VertexShader)

	camv := uset.AddStruct("Camera", vgpu.Float32Matrix4.Bytes()*3, 1, vgpu.Uniform, vgpu.VertexShader)

	txidxv := pcset.Add("TexIndex", vgpu.Int32, 1, vgpu.Push, vgpu.FragmentShader)
	tximgv := txset.Add("TexSampler", vgpu.ImageRGBA32, 1, vgpu.TextureRole, vgpu.FragmentShader)

	vset.ConfigValues(1) // val per var
	uset.ConfigValues(1)
	txset.ConfigValues(3)

	imgFiles := []string{"ground.png", "wood.png", "teximg.jpg"}
	imgs := make([]image.Image, len(imgFiles))
	for i, fnm := range imgFiles {
		pnm := filepath.Join("../images", fnm)
		imgs[i] = OpenImage(pnm)
		img, _ := tximgv.Values.ValueByIndexTry(i)
		img.Texture.ConfigGoImage(imgs[i].Bounds().Size(), 0)
		// img.Texture.Sampler.Border = vgpu.BorderBlack
		// img.Texture.Sampler.UMode = vgpu.ClampToBorder
		// img.Texture.Sampler.VMode = vgpu.ClampToBorder
	}

	sy.Config() // allocates everything etc

	// note: first val in set is offset
	rectPos, _ := posv.Values.ValueByIndexTry(0)
	rectPosA := rectPos.Floats32()
	rectPosA.Set(0,
		-0.5, -0.5, 0.0,
		0.5, -0.5, 0.0,
		0.5, 0.5, 0.0,
		-0.5, 0.5, 0.0)
	rectPos.SetMod()

	rectClr, _ := clrv.Values.ValueByIndexTry(0)
	rectClrA := rectClr.Floats32()
	rectClrA.Set(0,
		1.0, 0.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0,
		1.0, 1.0, 0.0)
	rectClr.SetMod()

	rectTex, _ := txcv.Values.ValueByIndexTry(0)
	rectTexA := rectTex.Floats32()
	rectTexA.Set(0,
		1.0, 0.0,
		0.0, 0.0,
		0.0, 1.0,
		1.0, 1.0)
	rectTex.SetMod()

	rectIndex, _ := idxv.Values.ValueByIndexTry(0)
	idxs := []uint16{0, 1, 2, 0, 2, 3}
	rectIndex.CopyFromBytes(unsafe.Pointer(&idxs[0]))

	for i, gimg := range imgs {
		img, _ := tximgv.Values.ValueByIndexTry(i)
		img.SetGoImage(gimg, 0, vgpu.FlipY)
	}

	// This is the standard camera view projection computation
	cam, _ := camv.Values.ValueByIndexTry(0)
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

	cam.CopyFromBytes(unsafe.Pointer(&camo)) // sets mod

	sy.Mem.SyncToGPU()

	vars.BindVarsStart(0) // only one set of bindings
	vars.BindStatVars(1)  // gets images
	vars.BindVarsEnd()

	vars.BindDynamicValue(0, camv, cam)

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		camo.Model.SetRotationY(.002 * float32(frameCount))
		cam.CopyFromBytes(unsafe.Pointer(&camo)) // sets mod
		sy.Mem.SyncToGPU()

		imgIndex := int32(frameCount % len(imgs))

		idx, ok := sf.AcquireNextImage()
		if !ok {
			return
		}

		cmd := sy.CmdPool.Buff
		descIndex := 0 // if running multiple frames in parallel, need diff sets

		sy.ResetBeginRenderPass(cmd, sf.Frames[idx], descIndex)
		pl.Push(cmd, txidxv, unsafe.Pointer(&imgIndex))
		pl.BindDrawVertex(cmd, descIndex)
		sy.EndRenderPass(cmd)

		sf.SubmitRender(cmd) // this is where it waits for the 16 msec
		sf.PresentImage(idx)
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
			if window.ShouldClose() {
				exitC <- struct{}{}
				continue
			}
			glfw.PollEvents()
			renderFrame()
		}
	}
}
