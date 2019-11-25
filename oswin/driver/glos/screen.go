// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glos

import (
	"image"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/window"
)

// This is called when a monitor is connected to or
// disconnected from the system.
func monitorChange(monitor *glfw.Monitor, event glfw.PeripheralEvent) {
	theApp.getScreens()
}

func (app *appImpl) getScreens() {
	glfw.SetMonitorCallback(monitorChange)
	mons := glfw.GetMonitors()
	sz := len(mons)
	if sz == 0 {
		app.noScreens = true
		// log.Printf("glos getScreens: no screens found!\n")
		return
	}
	app.mu.Lock()
	app.noScreens = false
	if len(app.screens) != sz {
		app.screens = make([]*oswin.Screen, sz)
	}
	for i := 0; i < sz; i++ {
		mon := mons[i]
		sc := app.screens[i]
		if sc == nil {
			sc = &oswin.Screen{}
			app.screens[i] = sc
		}
		if sc.Name == mon.GetName() {
			continue
		}
		pw, ph := mon.GetPhysicalSize()
		if pw == 0 {
			pw = 1024
		}
		if ph == 0 {
			ph = 768
		}
		x, y := mon.GetPos()
		vm := mon.GetVideoMode()
		cscx, _ := mon.GetContentScale() // note: requires glfw 3.3
		sc.Name = mon.GetName()
		sc.ScreenNumber = i
		sc.Geometry = image.Rectangle{Min: image.Point{x, y}, Max: image.Point{x + vm.Width, y + vm.Height}}
		depth := vm.RedBits + vm.GreenBits + vm.BlueBits
		sc.Depth = depth
		sc.PhysicalSize = image.Point{pw, ph}
		dpi := 25.4 * float32(vm.Width) / float32(pw)
		sc.PhysicalDPI = dpi
		sc.LogicalDPI = dpi
		sc.DevicePixelRatio = cscx
		sc.RefreshRate = float32(vm.RefreshRate)
	}
	if len(app.winlist) > 0 {
		fw := app.winlist[0]
		app.mu.Unlock()
		// fmt.Printf("sending screen update\n")
		fw.sendWindowEvent(window.ScreenUpdate)
	} else {
		app.mu.Unlock()
	}
}
