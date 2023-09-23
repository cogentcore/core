// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package driver

import (
	"fmt"
	"image"
	"testing"
	"time"

	"goki.dev/goosi"
	"goki.dev/vgpu/v2/vgpu"
)

func TestMain(t *testing.T) {
	Main(func(a goosi.App) {
		opts := &goosi.NewWindowOptions{
			Size:      image.Pt(1024, 768),
			StdPixels: true,
			Title:     "Goosi Test Window",
		}
		w, err := goosi.TheApp.NewWindow(opts)
		if err != nil {
			t.Error(err)
		}

		sy := w.Drawer().Sys
		sf := w.Drawer().Surf

		pl := sy.NewPipeline("drawtri")
		sy.ConfigRender(&sf.Format, vgpu.UndefType)
		sf.SetRender(&sy.Render)

		pl.AddShaderFile("trianglelit", vgpu.VertexShader, "testdata/trianglelit.spv")
		pl.AddShaderFile("vtxcolor", vgpu.FragmentShader, "testdata/vtxcolor.spv")

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
			exitC := make(chan struct{}, 2)

			fpsDelay := time.Second / 600
			fpsTicker := time.NewTicker(fpsDelay)
			for {
				select {
				case <-exitC:
					fpsTicker.Stop()
					destroy()
					return
				case <-fpsTicker.C:
					// if w.ShouldClose() {
					// 	exitC <- struct{}{}
					// 	continue
					// }
					// glfw.PollEvents()
					renderFrame()
				}
			}
		}
	})
}
