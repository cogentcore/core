// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build 3d

package glos

import (
	"image"

	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/goki/gi/oswin"
)

// This is called when a monitor is connected to or
// disconnected from the system.
func monitorChange(monitor *glfw.Monitor, event glfw.MonitorEvent) {
	// todo: could update more strategically
	theApp.getScreens()
}

func (app *appImpl) getScreens() {
	glfw.SetMonitorCallback(monitorChange)
	mons := glfw.GetMonitors()
	sz := len(mons)
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
		pw, ph := mon.GetPhysicalSize()
		x, y := mon.GetPos()
		vm := mon.GetVideoMode()
		sc.Name = mon.GetName()
		sc.ScreenNumber = i
		sc.Geometry = image.Rectangle{Min: image.Point{x, y}, Max: image.Point{x + vm.Width, y + vm.Height}}
		depth := vm.RedBits + vm.GreenBits + vm.BlueBits
		sc.Depth = depth
		sc.PhysicalSize = image.Point{pw, ph}
		dpi := 25.4 * float32(vm.Width) / float32(pw)
		sc.PhysicalDPI = dpi
		sc.LogicalDPI = dpi
		// todo: mac device pixel ratio!
		sc.RefreshRate = float32(vm.RefreshRate)
	}
}
