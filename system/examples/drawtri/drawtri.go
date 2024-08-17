// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"
	"fmt"
	"image"
	"time"

	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/system"
	_ "cogentcore.org/core/system/driver"
	"github.com/cogentcore/webgpu/wgpu"
)

//go:embed trianglelit.wgsl
var trianglelit string

func main() {
	opts := &system.NewWindowOptions{
		Size:      image.Pt(1024, 768),
		StdPixels: true,
		Title:     "System Draw Triangle",
	}
	w, err := system.TheApp.NewWindow(opts)
	if err != nil {
		panic(err)
	}

	system.TheApp.Cursor(w).SetSize(32)

	var sf *gpu.Surface
	var sy *gpu.GraphicsSystem
	var pl *gpu.GraphicsPipeline

	make := func() {
		sf = w.Drawer().Renderer().(*gpu.Surface)
		sy = gpu.NewGraphicsSystem(sf.GPU, "drawtri", sf)
		destroy := func() {
			sy.Release()
		}
		w.SetDestroyGPUResourcesFunc(destroy)

		pl = sy.AddGraphicsPipeline("drawtri")
		pl.SetFrontFace(wgpu.FrontFaceCW)

		sh := pl.AddShader("trianglelit")
		sh.OpenCode(trianglelit)
		pl.AddEntry(sh, gpu.VertexShader, "vs_main")
		pl.AddEntry(sh, gpu.FragmentShader, "fs_main")

		sy.Config()

		fmt.Println("made and configured pipelines")
	}

	frameCount := 0
	cur := cursors.Arrow
	stTime := time.Now()

	renderFrame := func() {
		if sf == nil {
			make()
		}
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()

		rp, err := sy.BeginRenderPass()
		if err != nil {
			return
		}
		pl.BindPipeline(rp)
		rp.Draw(3, 1, 0, 0)
		rp.End()
		sy.EndRenderPass(rp)

		frameCount++
		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 10 {
			fps := float64(frameCount) / dur
			fmt.Printf("fps: %.0f\n", fps)
			frameCount = 0
			stTime = eTime
		}
		if frameCount%60 == 0 {
			cur++
			if cur >= cursors.CursorN {
				cur = cursors.Arrow
			}
			err := system.TheApp.Cursor(w).Set(cur)
			if err != nil {
				fmt.Println("error setting cursor:", err)
			}
		}
	}

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
				// fmt.Println("paint")
				renderFrame()
			}
		}
	}()
	system.TheApp.MainLoop()
}
