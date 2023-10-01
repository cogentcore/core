// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"
	"fmt"
	"image"
	"time"

	"goki.dev/cursors"
	"goki.dev/goosi"
	"goki.dev/goosi/driver"
	"goki.dev/goosi/events"
	"goki.dev/vgpu/v2/vgpu"
)

//go:embed *.spv
var content embed.FS

func main() { driver.Main(mainrun) }

func mainrun(a goosi.App) {
	opts := &goosi.NewWindowOptions{
		Size:      image.Pt(1024, 768),
		StdPixels: true,
		Title:     "Goosi Test Window",
	}
	w, err := goosi.TheApp.NewWindow(opts)
	if err != nil {
		panic(err)
	}

	fmt.Println("got new window", w)

	goosi.TheApp.Cursor(w).SetSize(32)

	// note: drawer is always created and ready to go
	// we are creating an additional rendering system here.
	sf := w.Drawer().Surf
	sy := sf.GPU.NewGraphicsSystem("drawidx", &sf.Device)

	destroy := func() {
		sy.Destroy()
	}
	w.SetDestroyGPUResourcesFunc(destroy)

	pl := sy.NewPipeline("drawtri")
	sy.ConfigRender(&sf.Format, vgpu.UndefType)
	sf.SetRender(&sy.Render)

	pl.AddShaderEmbed("trianglelit", vgpu.VertexShader, content, "trianglelit.spv")
	pl.AddShaderEmbed("vtxcolor", vgpu.FragmentShader, content, "vtxcolor.spv")

	sy.Config()

	fmt.Println("made and configured pipelines")

	frameCount := 0
	cur := cursors.Default
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		idx := sf.AcquireNextImage()
		// fmt.Printf("\nacq: %v\n", time.Now().Sub(rt))
		descIdx := 0 // if running multiple frames in parallel, need diff sets
		cmd := sy.CmdPool.Buff
		sy.ResetBeginRenderPass(cmd, sf.Frames[idx], descIdx)
		// fmt.Printf("rp: %v\n", time.Now().Sub(rt))
		pl.BindPipeline(cmd)
		pl.Draw(cmd, 3, 1, 0, 0)
		sy.EndRenderPass(cmd)
		sf.SubmitRender(cmd) // this is where it waits for the 16 msec
		// fmt.Printf("submit %v\n", time.Now().Sub(rt))
		sf.PresentImage(idx)
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
				cur = cursors.Default
			}
			err := goosi.TheApp.Cursor(w).Set(cur)
			if err != nil {
				fmt.Println("error setting cursor:", err)
			}
		}
	}

	for {
		fmt.Println("waiting for event")
		evi := w.NextEvent()
		fmt.Println("got event", evi)
		et := evi.Type()
		switch et {
		case events.Window:
			ev := evi.(*events.WindowEvent)
			switch ev.Action {
			case events.Close:
				return
			}
		case events.WindowPaint:
			// fmt.Println("paint")
			renderFrame()
		case events.MouseMove:
			// fmt.Println("got mouse event at pos", evi.Pos())
		}
	}
}
