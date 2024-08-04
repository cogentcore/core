// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"time"
	"unsafe"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	_ "cogentcore.org/core/system/driver"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

//go:embed indexed.wgsl
var indexed string

type CamView struct {
	Model      math32.Matrix4
	View       math32.Matrix4
	Projection math32.Matrix4
}

func main() {
	opts := &system.NewWindowOptions{
		Size:      image.Pt(1024, 768),
		StdPixels: true,
		Title:     "System Test Window",
	}
	w, err := system.TheApp.NewWindow(opts)
	if err != nil {
		panic(err)
	}
	// w.SetFPS(20) // 60 default

	var sf *gpu.Surface
	var sy *gpu.System
	var pl *gpu.GraphicsPipeline
	var cam *gpu.Value
	var camo CamView

	make := func() {
		// note: drawer is always created and ready to go
		// we are creating an additional rendering system here.
		sf = w.Drawer().Surface().(*gpu.Surface)
		sy = sf.GPU.NewGraphicsSystem("drawidx", sf.Device)

		destroy := func() {
			sy.Release()
		}
		w.SetDestroyGPUResourcesFunc(destroy)

		pl = sy.AddGraphicsPipeline("drawidx")
		pl.SetCullMode(wgpu.CullModeNone)
		sy.SetClearColor(color.RGBA{50, 50, 50, 255})

		sh := pl.AddShader("indexed")
		sh.OpenCode(indexed)
		pl.AddEntry(sh, gpu.VertexShader, "vs_main")
		pl.AddEntry(sh, gpu.FragmentShader, "fs_main")

		vgp := sy.Vars.AddVertexGroup()
		ugp := sy.Vars.AddGroup(gpu.Uniform)

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
		cam = camv.Values.Values[0]
		campos := math32.Vec3(0, 0, 2)
		target := math32.Vec3(0, 0, 0)
		var lookq math32.Quat
		lookq.SetFromRotationMatrix(math32.NewLookAt(campos, target, math32.Vec3(0, 1, 0)))
		scale := math32.Vec3(1, 1, 1)
		var cview math32.Matrix4
		cview.SetTransform(campos, lookq, scale)
		view, _ := cview.Inverse()

		camo.Model.SetIdentity()
		camo.View.CopyFrom(view)
		aspect := float32(sf.Format.Size.X) / float32(sf.Format.Size.Y)
		fmt.Printf("aspect: %g\n", aspect)
		camo.Projection.SetPerspective(45, aspect, 0.01, 100)
		gpu.SetValueFrom(cam, []CamView{camo}) // note: always use slice to copy

		fmt.Println("made and configured pipelines")
	}

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		if sf == nil {
			make()
		}
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		camo.Model.SetRotationY(.1 * float32(frameCount))
		gpu.SetValueFrom(cam, []CamView{camo})

		view, err := sf.AcquireNextTexture()
		if errors.Log(err) != nil {
			return
		}
		cmd := sy.NewCommandEncoder()
		rp := sy.BeginRenderPass(cmd, view)
		pl.BindPipeline(rp)
		pl.BindDrawIndexed(rp)
		rp.End()
		sf.SubmitRender(rp, cmd)
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

	haveKeyboard := false
	go func() {
		for {
			evi := w.Events().Deque.NextEvent()
			et := evi.Type()
			if et != events.WindowPaint && et != events.MouseMove {
				fmt.Println("got event", evi)
			}
			switch et {
			case events.Window:
				ev := evi.(*events.WindowEvent)
				fmt.Println("got window event", ev)
				switch ev.Action {
				case events.WinShow:
					make()
				case events.WinClose:
					fmt.Println("got events.Close; quitting")
					system.TheApp.Quit()
				}
			case events.WindowPaint:
				if w.IsVisible() {
					renderFrame()
				} else {
					fmt.Println("skipping paint event")
				}
			case events.MouseDown:
				if haveKeyboard {
					system.TheApp.HideVirtualKeyboard()
				} else {
					system.TheApp.ShowVirtualKeyboard(styles.KeyboardMultiLine)
				}
				haveKeyboard = !haveKeyboard
			}
		}
	}()
	system.TheApp.MainLoop()
}
