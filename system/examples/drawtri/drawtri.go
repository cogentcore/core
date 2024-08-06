// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"
	"fmt"
	"image"
	"time"

	"cogentcore.org/core/base/errors"
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
		Title:     "System Test Window",
	}
	w, err := system.TheApp.NewWindow(opts)
	if err != nil {
		panic(err)
	}

	fmt.Println("got new window", w)

	system.TheApp.Cursor(w).SetSize(32)

	var sf *gpu.Surface
	var sy *gpu.System
	var pl *gpu.GraphicsPipeline

	make := func() {
		// note: drawer is always created and ready to go
		// we are creating an additional rendering system here.
		sf = w.Drawer().Surface().(*gpu.Surface)
		sy = sf.GPU.NewGraphicsSystem("drawtri", sf.Device)
		destroy := func() {
			sy.Release()
		}
		w.SetDestroyGPUResourcesFunc(destroy)

		pl = sy.AddGraphicsPipeline("drawtri")
		sy.ConfigRender(&sf.Format, gpu.UndefinedType, sf)
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
		view, err := sf.AcquireNextTexture()
		if errors.Log(err) != nil {
			return
		}
		// fmt.Printf("\nacq: %v\n", time.Now().Sub(rt))
		cmd := sy.NewCommandEncoder()
		rp := sy.BeginRenderPass(cmd, view)
		// fmt.Printf("rp: %v\n", time.Now().Sub(rt))
		pl.BindPipeline(rp)
		rp.Draw(3, 1, 0, 0)
		rp.End()
		sf.SubmitRender(rp, cmd) // this is where it waits for the 16 msec
		sf.Present()
		// fmt.Printf("submit %v\n", time.Now().Sub(rt))
		// fmt.Printf("present %v\n\n", time.Now().Sub(rt))

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
