// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"time"

	"goki.dev/goosi"
	"goki.dev/goosi/driver"
	"goki.dev/goosi/mouse"
	"goki.dev/goosi/window"
	"goki.dev/vgpu/v2/vgpu"
)

func main() {
	driver.Main(func(a goosi.App) {
		opts := &goosi.NewWindowOptions{
			Size:      image.Pt(1024, 768),
			StdPixels: true,
			Title:     "Goosi Test Window",
		}
		w, err := goosi.TheApp.NewWindow(opts)
		if err != nil {
			panic(err)
		}

		sy := w.Drawer().Sys
		sf := w.Drawer().Surf

		pl := sy.NewPipeline("drawtri")
		sy.ConfigRender(&sf.Format, vgpu.UndefType)
		sf.SetRender(&sy.Render)

		pl.AddShaderFile("trianglelit", vgpu.VertexShader, "trianglelit.spv")
		pl.AddShaderFile("vtxcolor", vgpu.FragmentShader, "vtxcolor.spv")

		sy.Config()

		destroy := func() {
			sy.Destroy()
			sf.Destroy()
			vgpu.Terminate()
		}

		frameCount := 0
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
		}

		for {
			evi := w.NextEvent()
			fmt.Println("got event", evi)
			switch ev := evi.(type) {
			case *window.Event:
				switch ev.Action {
				case window.Close:
					destroy()
					return
				case window.Paint:
					renderFrame()
				}
			case *mouse.Event:
				fmt.Println("got mouse event at pos", ev.Pos())
			}
		}
	})
}
