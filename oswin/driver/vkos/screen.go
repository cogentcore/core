// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vkos

import (
	"image"
	"log"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
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
		log.Printf("vkos monitorChange: %v event: %v\n", monitor.GetName(), enm)
	}
	theApp.GetScreens()
}

func (app *appImpl) GetScreens() {
	app.mu.Lock()
	mons := glfw.GetMonitors()
	sz := len(mons)
	if sz == 0 {
		app.noScreens = true
		if monitorDebug {
			log.Printf("vkos getScreens: no screens found!\n")
		}
		app.mu.Unlock()
		return
	}
	if monitorDebug {
		pm := glfw.GetPrimaryMonitor()
		log.Printf("Primary monitor: %s   first monitor: %s\n", pm.GetName(), mons[0].GetName())
	}
	app.noScreens = false
	gotNew := sz != len(app.screens)
	if gotNew {
		app.screens = make([]*oswin.Screen, 0, sz)
	}
	for i := 0; i < sz; i++ {
		mon := mons[i]
		if monitorDebug {
			log.Printf("vkos getScreens: mon number: %v name: %v\n", i, mon.GetName())
		}
		if len(app.screens) <= i {
			app.screens = append(app.screens, &oswin.Screen{})
		}
		sc := app.screens[i]
		vm := mon.GetVideoMode()
		if vm.Width == 0 || vm.Height == 0 {
			if monitorDebug {
				log.Printf("vkos getScreens: screen %v has no size -- skipping\n", sc.Name)
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
		if mat32.IsNaN(cscx) {
			log.Printf("GetContentScale returned nan -- not good..\n")
			cscx = 1
		}
		if cscx < 1 {
			cscx = 1
		}
		sc.Name = mon.GetName()
		sc.ScreenNumber = i
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
		if monitorDebug {
			log.Printf("screen %d:\n%s\n", i, kit.StringJSON(sc))
		}
	}
	if gotNew && len(app.winlist) > 0 {
		fw := app.winlist[0]
		app.mu.Unlock()
		if monitorDebug {
			log.Printf("vkos getScreens: sending screen update\n")
		}
		fw.sendWindowEvent(window.ScreenUpdate)
	} else {
		if monitorDebug {
			log.Printf("vkos getScreens: no screen changes, NOT sending screen update\n")
		}
		app.mu.Unlock()
	}
}
