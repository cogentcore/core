// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glos

import (
	"image"
	"log"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/window"
)

// monitorDebug turns on various debugging statements about monitor changes
// and updates from glfw.
var monitorDebug = false

// This is called when a monitor is connected to or
// disconnected from the system.
func monitorChange(monitor *glfw.Monitor, event glfw.PeripheralEvent) {
	if monitorDebug {
		enm := ""
		if event == glfw.Connected {
			enm = "Connected"
		} else {
			enm = "Disconnected"
		}
		log.Printf("glos monitorChange: %v event: %v\n", monitor.GetName(), enm)
	}
	theApp.getScreens()
}

func (app *appImpl) getScreens() {
	mons := glfw.GetMonitors()
	sz := len(mons)
	if sz == 0 {
		app.noScreens = true
		if monitorDebug {
			log.Printf("glos getScreens: no screens found!\n")
		}
		return
	}
	app.mu.Lock()
	app.noScreens = false
	gotNew := false
	for i := 0; i < sz; i++ {
		mon := mons[i]
		if monitorDebug {
			log.Printf("glos getScreens: mon number: %v name: %v\n", i, mon.GetName())
		}
		var sc *oswin.Screen
		var sci int
		for j, scc := range app.screens {
			if scc != nil && scc.Name == mon.GetName() {
				sc = scc
				sci = j
				break
			}
		}
		if sc == nil {
			gotNew = true
			sc = &oswin.Screen{}
			sci = len(app.screens)
			app.screens = append(app.screens, sc)
		}
		vm := mon.GetVideoMode()
		if vm.Width == 0 || vm.Height == 0 {
			if monitorDebug {
				log.Printf("glos getScreens: screen %v has no size -- skipping\n", sc.Name)
			}
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
		cscx, _ := mon.GetContentScale() // note: requires glfw 3.3
		if cscx < 1 {
			cscx = 1
		}
		sc.Name = mon.GetName()
		sc.ScreenNumber = sci
		sc.Geometry = image.Rectangle{Min: image.Point{x, y}, Max: image.Point{x + vm.Width, y + vm.Height}}
		sc.DevicePixelRatio = cscx
		sc.PixSize.X = int(float32(vm.Width) * cscx)
		sc.PixSize.Y = int(float32(vm.Height) * cscx)
		depth := vm.RedBits + vm.GreenBits + vm.BlueBits
		sc.Depth = depth
		sc.PhysicalSize = image.Point{pw, ph}
		dpi := 25.4 * float32(sc.PixSize.X) / float32(pw)
		sc.PhysicalDPI = dpi
		if sc.LogicalDPI == 0 { // do not overwrite if already set
			sc.LogicalDPI = dpi
		}
		sc.RefreshRate = float32(vm.RefreshRate)
	}
	if gotNew && len(app.winlist) > 0 {
		fw := app.winlist[0]
		app.mu.Unlock()
		if monitorDebug {
			log.Printf("glos getScreens: sending screen update\n")
		}
		fw.sendWindowEvent(window.ScreenUpdate)
	} else {
		app.mu.Unlock()
	}
}
