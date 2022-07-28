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
	app.screens = make([]*oswin.Screen, 0, sz)
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
				log.Printf("vkos getScreens: screen %v has no size!\n", sc.Name)
			}
			if app.Platform() == oswin.MacOS {
				si, has := app.findScreenInfo("Built-in Retina Display")
				if has {
					*sc = *si
					sc.ScreenNumber = i
					if monitorDebug {
						log.Printf("vkos getScreens: MacOS recovered screen info from %v\n", sc.Name)
					}
				} else { // use plausible defaults.. sheesh
					sc.Name = "Built-in Retina Display"
					sc.Geometry.Max = image.Point{1728, 1117}
					sc.DevicePixelRatio = 2
					sc.PixSize = sc.Geometry.Max.Mul(2)
					sc.PhysicalSize = image.Point{344, 222}
					sc.PhysicalDPI = 255.1814
					sc.LogicalDPI = 255.1814
					sc.Depth = 24
					sc.RefreshRate = 60
					if monitorDebug {
						log.Printf("vkos getScreens: MacOS unknown display set to Built-in Retina Display %d:\n%s\n", i, kit.StringJSON(sc))
					}
				}
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
			if monitorDebug {
				log.Printf("GetContentScale on %s returned nan -- trying saved info..\n", mon.GetName())
			}
			si, has := app.findScreenInfo(mon.GetName())
			if has {
				cscx = si.DevicePixelRatio
				if monitorDebug {
					log.Printf("recovered value of: %g for screen: %s\n", cscx, si.Name)
				}
			} else {
				cscx = 1
				if monitorDebug {
					log.Printf("using default of 1 -- may not be correct!\n")
				}
			}
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
		app.saveScreenInfo(sc)
	}
	if len(app.winlist) > 0 {
		fw := app.winlist[0]
		app.mu.Unlock()
		if monitorDebug {
			log.Printf("vkos getScreens: sending screen update\n")
		}
		fw.sendWindowEvent(window.ScreenUpdate)
	} else {
		if monitorDebug {
			log.Printf("vkos getScreens: no windows, NOT sending screen update\n")
		}
		app.mu.Unlock()
	}
}

// saveScreenInfo saves a copy of given screen info to screensAll list if unique
// based on name.  Returns true if added a new screen.
func (app *appImpl) saveScreenInfo(sc *oswin.Screen) bool {
	_, has := app.findScreenInfo(sc.Name)
	if has {
		return false
	}
	nsc := &oswin.Screen{}
	*nsc = *sc
	app.screensAll = append(app.screensAll, nsc)
	return true
}

// findScreenInfo finds saved screen info based on name
func (app *appImpl) findScreenInfo(name string) (*oswin.Screen, bool) {
	for _, sc := range app.screensAll {
		if sc.Name == name {
			return sc, true
		}
	}
	return nil, false
}
