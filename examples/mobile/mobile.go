package main

import (
	"embed"
	"log"
	"time"

	"github.com/goki/mobile/app"
	"github.com/goki/mobile/event/lifecycle"
	"github.com/goki/mobile/event/paint"
	"github.com/goki/mobile/event/size"
	"github.com/goki/mobile/event/touch"
	"github.com/goki/vgpu/vgpu"
	vk "github.com/goki/vulkan"
)

//go:embed *.spv
var content embed.FS

var (
	gpu      *vgpu.GPU
	system   *vgpu.System
	surface  *vgpu.Surface
	pipeline *vgpu.Pipeline
	window   uintptr

	frameCount int
	stTime     time.Time
	fpsDelay   time.Duration
	fpsTicker  *time.Ticker

	touchX float32
	touchY float32
)

func onStart(a app.App) {
	winext := vk.GetRequiredInstanceExtensions()
	log.Printf("required exts: %#v\n", winext)
	gpu = vgpu.NewGPU()
	gpu.AddInstanceExt(winext...)
	gpu.Config("drawtri")

	var sf vk.Surface
	log.Println("in onStart", gpu.Instance, window, &sf)
	ret := vk.CreateWindowSurface(gpu.Instance, window, nil, &sf)
	if err := vk.Error(ret); err != nil {
		log.Println("vulkan error:", err)
		return
	}
	surface = vgpu.NewSurface(gpu, sf)

	log.Printf("format: %s\n", surface.Format.String())

	system = gpu.NewGraphicsSystem("drawtri", &surface.Device)
	pipeline = system.NewPipeline("drawtri")
	system.ConfigRender(&surface.Format, vgpu.UndefType)
	surface.SetRender(&system.Render)

	pipeline.AddShaderEmbed("trianglelit", vgpu.VertexShader, content, "trianglelit.spv")
	pipeline.AddShaderEmbed("vtxcolor", vgpu.FragmentShader, content, "vtxcolor.spv")

	system.Config()

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
	system.Destroy()
	system = nil
	surface.Destroy()
	gpu.Destroy()
	vgpu.Terminate()
}

func onPaint() {
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
			log.Printf("fps: %.0f\n", fps)
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
