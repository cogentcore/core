package main

import (
	"embed"
	"fmt"
	"log"
	"time"

	"github.com/goki/vgpu/vgpu"
	vk "github.com/goki/vulkan"
	"github.com/xlab/android-go/android"
	"github.com/xlab/android-go/app"
)

//go:embed *.spv
var content embed.FS

func init() {
	app.SetLogTag("VulkanCube")
}

func main() {
	nativeWindowEvents := make(chan app.NativeWindowEvent)
	inputQueueEvents := make(chan app.InputQueueEvent, 1)
	inputQueueChan := make(chan *android.InputQueue, 1)

	app.Main(func(a app.NativeActivity) {
		// disable this to get the stack
		// defer catcher.Catch(
		// 	catcher.RecvLog(true),
		// 	catcher.RecvDie(-1),
		// )

		vgpu.Debug = true

		// orPanic(vk.SetDefaultGetInstanceProcAddr())
		orPanic(vk.Init())
		a.HandleNativeWindowEvents(nativeWindowEvents)
		a.HandleInputQueueEvents(inputQueueEvents)
		// just skip input events (so app won't be dead on touch input)
		go app.HandleInputQueues(inputQueueChan, func() {
			a.InputQueueHandled()
		}, app.SkipInputEvents)
		a.InitDone()

		var (
			gpu      *vgpu.GPU
			system   *vgpu.System
			surface  *vgpu.Surface
			pipeline *vgpu.Pipeline
			window   uintptr
		)

		frameCount := 0
		stTime := time.Now()
		fpsDelay := time.Second / 600
		fpsTicker := time.NewTicker(fpsDelay)
		for {
			select {
			case <-a.LifecycleEvents():
				// ignore
			case event := <-inputQueueEvents:
				switch event.Kind {
				case app.QueueCreated:
					inputQueueChan <- event.Queue
				case app.QueueDestroyed:
					inputQueueChan <- nil
				}
			case event := <-nativeWindowEvents:
				switch event.Kind {
				case app.NativeWindowCreated:

					winext := vk.GetRequiredInstanceExtensions()
					log.Printf("required exts: %#v\n", winext)
					gpu = vgpu.NewGPU()
					gpu.AddInstanceExt(winext...)
					vgpu.Debug = true
					gpu.Config("drawtri")
					window = event.Window.Ptr()

					var sf vk.Surface
					ret := vk.CreateWindowSurface(gpu.Instance, window, nil, &sf)
					if err := vk.Error(ret); err != nil {
						log.Println("vulkan error:", err)
						break
					}
					surface = vgpu.NewSurface(gpu, sf)

					fmt.Printf("format: %s\n", surface.Format.String())

					system = gpu.NewGraphicsSystem("drawtri", &surface.Device)
					pipeline = system.NewPipeline("drawtri")
					system.ConfigRender(&surface.Format, vgpu.UndefType)
					surface.SetRender(&system.Render)

					pipeline.AddShaderEmbed("trianglelit", vgpu.VertexShader, content, "trianglelit.spv")
					pipeline.AddShaderEmbed("vtxcolor", vgpu.FragmentShader, content, "vtxcolor.spv")

					system.Config()

				case app.NativeWindowDestroyed:
					vk.DeviceWaitIdle(surface.Device.Device)
					system.Destroy()
					system = nil
					surface.Destroy()
					gpu.Destroy()
					vgpu.Terminate()
				case app.NativeWindowRedrawNeeded:
					a.NativeWindowRedrawDone()
				}
			case <-fpsTicker.C:
				if system != nil {
					idx := surface.AcquireNextImage()
					// fmt.Printf("\nacq: %v\n", time.Now().Sub(rt))
					descIdx := 0 // if running multiple frames in parallel, need diff sets
					cmd := system.CmdPool.Buff
					system.ResetBeginRenderPass(cmd, surface.Frames[idx], descIdx)
					// fmt.Printf("rp: %v\n", time.Now().Sub(rt))
					pipeline.BindPipeline(cmd)
					pipeline.Draw(cmd, 3, 1, 0, 0)
					system.EndRenderPass(cmd)
					surface.SubmitRender(cmd) // this is where it waits for the 16 msec
					// fmt.Printf("submit %v\n", time.Now().Sub(rt))
					surface.PresentImage(idx)

					frameCount++
					eTime := time.Now()
					dur := float64(eTime.Sub(stTime)) / float64(time.Second)
					if dur > 10 {
						fps := float64(frameCount) / dur
						fmt.Printf("fps: %.0f\n", fps)
						frameCount = 0
						stTime = eTime
					}

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
