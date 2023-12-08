// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"image"
	"log"

	"github.com/go-gl/glfw/v3.3/glfw"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
)

// MonitorDebug turns on various debugging statements about monitor changes
// and updates from glfw.
var MonitorDebug = false

// Note: MacOS monitor situation is significantly buggy.  we try to work around this.
// https://github.com/glfw/glfw/issues/2160

var MacOsBuiltinMonitor = "Built-in Retina Display"

// MonitorChange is called when a monitor is connected to or
// disconnected from the system.
func (a *App) MonitorChange(monitor *glfw.Monitor, event glfw.PeripheralEvent) {
	if MonitorDebug {
		enm := "Unknown"
		if event == glfw.Connected {
			enm = "Connected"
		} else {
			enm = "Disconnected"
		}
		log.Printf("MonitorDebug: monitorChange: %v event: %v\n", monitor.GetName(), enm)
	}
	a.GetScreens()
	if len(a.Windows) > 0 {
		fw := a.Windows[0]
		if MonitorDebug {
			log.Printf("MonitorDebug: monitorChange: sending screen update\n")
		}
		fw.EvMgr.Window(events.ScreenUpdate)
	} else {
		if MonitorDebug {
			log.Printf("MonitorDebug: monitorChange: no windows, NOT sending screen update\n")
		}
	}
}

func (a *App) GetScreens() {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	mons := glfw.GetMonitors()
	sz := len(mons)
	if sz == 0 {
		a.Screens = []*goosi.Screen{}
		if MonitorDebug {
			log.Printf("MonitorDebug: getScreens: no screens found!\n")
		}
		return
	}
	if MonitorDebug {
		pm := glfw.GetPrimaryMonitor()
		log.Printf("MonitorDebug: Primary monitor: %s   first monitor: %s\n", pm.GetName(), mons[0].GetName())
	}
	a.Screens = make([]*goosi.Screen, 0, sz)
	scNo := 0
	for i := 0; i < sz; i++ {
		mon := mons[i]
		if MonitorDebug {
			log.Printf("MonitorDebug: getScreens: mon number: %v name: %v\n", i, mon.GetName())
		}
		if len(a.Screens) <= scNo {
			a.Screens = append(a.Screens, &goosi.Screen{})
		}
		sc := a.Screens[scNo]
		vm := mon.GetVideoMode()
		if vm.Width == 0 || vm.Height == 0 {
			if MonitorDebug {
				log.Printf("MonitorDebug: getScreens: screen %v has no size!\n", sc.Name)
			}
			if a.Platform() == goosi.MacOS {
				si, has := a.FindScreenInfo(MacOsBuiltinMonitor)
				if has {
					*sc = *si
					sc.ScreenNumber = scNo
					sc.UpdateLogicalDPI()
					if MonitorDebug {
						log.Printf("MonitorDebug: getScreens: MacOS recovered screen info from %v\n", sc.Name)
					}
					scNo++
					continue
				} else { // use plausible defaults.. sheesh
					sc.ScreenNumber = scNo
					sc.Name = MacOsBuiltinMonitor
					sc.Geometry.Max = image.Point{2056, 1329}
					sc.DevicePixelRatio = 2
					sc.PixSize = sc.Geometry.Max.Mul(2)
					sc.PhysicalSize = image.Point{344, 222}
					sc.PhysicalDPI = 25.4 * float32(sc.PixSize.X) / float32(sc.PhysicalSize.X)
					sc.Depth = 24
					sc.RefreshRate = 60
					sc.UpdateLogicalDPI()
					if MonitorDebug {
						log.Printf("MonitorDebug: getScreens: MacOS unknown display set to Built-in Retina Display %d:\n%s\n", i, laser.StringJSON(sc))
					}
					scNo++
					continue
				}
			}
			a.Screens = a.Screens[0 : len(a.Screens)-1] // not all there
			continue
		}
		pw, ph := mon.GetPhysicalSize()
		if pw == 0 {
			if MonitorDebug {
				log.Printf("MonitorDebug: physical size %s returned 0 -- bailing\n", mon.GetName())
			}
			a.Screens = a.Screens[0 : len(a.Screens)-1] // not all there
		}
		x, y := mon.GetPos()
		cscx, _ := mon.GetContentScale() // note: requires glfw 3.3
		if mat32.IsNaN(cscx) {
			if MonitorDebug {
				log.Printf("MonitorDebug: GetContentScale on %s returned NaN -- trying saved info..\n", mon.GetName())
			}
			si, has := a.FindScreenInfo(mon.GetName())
			if has {
				cscx = si.DevicePixelRatio
				if MonitorDebug {
					log.Printf("MonitorDebug: recovered value of: %g for screen: %s\n", cscx, si.Name)
				}
			} else {
				cscx = 1
				if MonitorDebug {
					log.Printf("MonitorDebug: using default of 1 -- may not be correct!\n")
				}
			}
		}
		if cscx < 1 {
			cscx = 1
		}
		sc.Name = mon.GetName()
		sc.ScreenNumber = scNo
		sc.Geometry = image.Rectangle{Min: image.Point{x, y}, Max: image.Point{x + vm.Width, y + vm.Height}}
		sc.DevicePixelRatio = cscx
		sc.PixSize = sc.WinSizeToPix(image.Point{vm.Width, vm.Height})
		depth := vm.RedBits + vm.GreenBits + vm.BlueBits
		sc.Depth = depth
		sc.PhysicalSize = image.Point{pw, ph}
		dpi := 25.4 * float32(sc.PixSize.X) / float32(pw)
		sc.PhysicalDPI = dpi
		sc.UpdateLogicalDPI()
		sc.RefreshRate = float32(vm.RefreshRate)
		if MonitorDebug {
			log.Printf("MonitorDebug: screen %d:\n%s\n", scNo, laser.StringJSON(sc))
		}
		a.SaveScreenInfo(sc)
		scNo++
	}

	// if originally a non-builtin monitor was primary, and now builtin is primary,
	// then switch builtin and non-builtin.  see https://github.com/glfw/glfw/issues/2160
	if sz > 1 && a.Platform() == goosi.MacOS && a.Screens[0].Name == MacOsBuiltinMonitor {
		fss := a.AllScreens[0]
		if fss.Name != MacOsBuiltinMonitor {
			if MonitorDebug {
				log.Printf("MonitorDebug: getScreens: MacOs, builtin is currently primary, but was not originally -- restoring primary monitor as: %s\n", a.Screens[1].Name)
			}
			// assume 2nd one is good..
			a.Screens[0], a.Screens[1] = a.Screens[1], a.Screens[0] // swap
			a.Screens[0].ScreenNumber = 0
			a.Screens[1].ScreenNumber = 1
		}
	}
	a.Mu.Unlock()
}

// SaveScreenInfo saves a copy of given screen info to screensAll list if unique
// based on name. Returns true if added a new screen.
func (a *App) SaveScreenInfo(sc *goosi.Screen) bool {
	_, has := a.FindScreenInfo(sc.Name)
	if has {
		return false
	}
	nsc := &goosi.Screen{}
	*nsc = *sc
	a.AllScreens = append(a.AllScreens, nsc)
	return true
}

// FindScreenInfo finds saved screen info based on name
func (a *App) FindScreenInfo(name string) (*goosi.Screen, bool) {
	for _, sc := range a.AllScreens {
		if sc.Name == name {
			return sc, true
		}
	}
	return nil, false
}
