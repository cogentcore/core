// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"image"
	"log"
	"slices"
	"time"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/system"
	"github.com/go-gl/glfw/v3.3/glfw"
)

var (
	// ScreenDebug turns on various debugging statements about monitor changes
	// and updates from glfw.
	ScreenDebug = false

	// ScreenPollInterval is the time between checking if screens have changed.
	// This is primarily for detecting changes after sleep, in which case
	// the interval has likely been much longer than a few seconds.
	ScreenPollInterval = time.Second

	// Note: MacOS monitor situation is significantly buggy.  we try to work around this.
	// https://github.com/glfw/glfw/issues/2160
	MacOsBuiltinMonitor = "Built-in Retina Display"
)

// MonitorChange is called when a monitor is connected to or
// disconnected from the system.
func (a *App) MonitorChange(monitor *glfw.Monitor, event glfw.PeripheralEvent) {
	if ScreenDebug {
		enm := "Unknown"
		if event == glfw.Connected {
			enm = "Connected"
		} else {
			enm = "Disconnected"
		}
		log.Printf("ScreenDebug: monitorChange: %v event: %v\n", monitor.GetName(), enm)
	}
	a.GetScreens()
	if len(a.Windows) > 0 {
		fw := a.Windows[0]
		if ScreenDebug {
			log.Println("ScreenDebug: monitorChange: sending screen update")
		}
		fw.Event.Window(events.ScreenUpdate)
	} else {
		if ScreenDebug {
			log.Println("ScreenDebug: monitorChange: no windows, NOT sending screen update")
		}
	}
}

var lastScreenPoll time.Time

func (a *App) PollScreenChanges() {
	now := time.Now()
	if now.Sub(lastScreenPoll) < ScreenPollInterval {
		return
	}
	lastScreenPoll = now

	mons := glfw.GetMonitors()
	ns := len(mons)
	if ns == 0 {
		return
	}
	csc := make([]*system.Screen, ns)
	if len(mons) != len(a.Screens) {
		goto doUpdate
	}
	for i, sc := range a.Screens {
		ssc := &system.Screen{}
		*ssc = *sc
		csc[i] = ssc
	}
	a.GetScreens()
	for i, sc := range a.Screens {
		ssc := csc[i]
		if *ssc != *sc {
			goto doUpdate
		}
	}
	return // no change
doUpdate:
	if len(a.Windows) > 0 {
		fw := a.Windows[0]
		if ScreenDebug {
			log.Println("ScreenDebug: PollScreenChanges: sending screen update")
		}
		fw.Event.Window(events.ScreenUpdate)
	}
}

func (a *App) GetScreens() {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	mons := glfw.GetMonitors()
	a.Monitors = mons
	sz := len(mons)
	if sz == 0 {
		a.Screens = []*system.Screen{}
		if ScreenDebug {
			log.Printf("ScreenDebug: getScreens: no screens found!\n")
		}
		return
	}
	if ScreenDebug {
		pm := glfw.GetPrimaryMonitor()
		log.Printf("ScreenDebug: Primary monitor: %s   first monitor: %s\n", pm.GetName(), mons[0].GetName())
	}
	a.Screens = make([]*system.Screen, 0, sz)
	scNo := 0
	for i := 0; i < sz; i++ {
		mon := mons[i]
		if ScreenDebug {
			log.Printf("ScreenDebug: getScreens: mon number: %v name: %v\n", i, mon.GetName())
		}
		for len(a.Screens) <= scNo {
			a.Screens = append(a.Screens, &system.Screen{})
		}
		sc := a.Screens[scNo]
		vm := mon.GetVideoMode()
		if vm.Width == 0 || vm.Height == 0 {
			if ScreenDebug {
				log.Printf("ScreenDebug: getScreens: screen %v has no size!\n", sc.Name)
			}
			if a.Platform() == system.MacOS {
				si, has := a.FindScreenInfo(MacOsBuiltinMonitor)
				if has {
					*sc = *si
					sc.ScreenNumber = scNo
					sc.UpdateLogicalDPI()
					if ScreenDebug {
						log.Printf("ScreenDebug: getScreens: MacOS recovered screen info from %v\n", sc.Name)
					}
					scNo++
					continue
				}
				// use plausible defaults.. sheesh
				sc.ScreenNumber = scNo
				sc.Name = MacOsBuiltinMonitor
				sc.Geometry.Max = image.Point{2056, 1329}
				sc.DevicePixelRatio = 2
				sc.PixelSize = sc.Geometry.Max.Mul(2)
				sc.PhysicalSize = image.Point{344, 222}
				sc.UpdatePhysicalDPI()
				sc.Depth = 24
				sc.RefreshRate = 60
				sc.UpdateLogicalDPI()
				if ScreenDebug {
					log.Printf("ScreenDebug: getScreens: MacOS unknown display set to Built-in Retina Display %d:\n%s\n", i, reflectx.StringJSON(sc))
				}
				scNo++
				continue
			}
			a.Screens = slices.Delete(a.Screens, scNo, scNo+1)
			a.Monitors = slices.Delete(a.Monitors, scNo, scNo+1)
			continue
		}
		pw, ph := mon.GetPhysicalSize()
		if pw == 0 {
			if ScreenDebug {
				log.Printf("ScreenDebug: physical size %s returned 0 -- bailing\n", mon.GetName())
			}
			a.Screens = slices.Delete(a.Screens, scNo, scNo+1)
			a.Monitors = slices.Delete(a.Monitors, scNo, scNo+1)
			continue
		}
		x, y := mon.GetPos()
		cscx, _ := mon.GetContentScale() // note: requires glfw 3.3
		if math32.IsNaN(cscx) {
			if ScreenDebug {
				log.Printf("ScreenDebug: GetContentScale on %s returned NaN -- trying saved info..\n", mon.GetName())
			}
			si, has := a.FindScreenInfo(mon.GetName())
			if has {
				cscx = si.DevicePixelRatio
				if ScreenDebug {
					log.Printf("ScreenDebug: recovered value of: %g for screen: %s\n", cscx, si.Name)
				}
			} else {
				cscx = 1
				if ScreenDebug {
					log.Printf("ScreenDebug: using default of 1 -- may not be correct!\n")
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
		sc.PixelSize = sc.WindowSizeToPixels(image.Point{vm.Width, vm.Height})
		depth := vm.RedBits + vm.GreenBits + vm.BlueBits
		sc.Depth = depth
		sc.PhysicalSize = image.Point{pw, ph}
		sc.UpdatePhysicalDPI()
		sc.UpdateLogicalDPI()
		sc.RefreshRate = float32(vm.RefreshRate)
		if ScreenDebug {
			log.Printf("ScreenDebug: screen %d:\n%s\n", scNo, reflectx.StringJSON(sc))
		}
		a.SaveScreenInfo(sc)
		scNo++
	}

	// if originally a non-builtin monitor was primary, and now builtin is primary,
	// then switch builtin and non-builtin.  see https://github.com/glfw/glfw/issues/2160
	if sz > 1 && a.Platform() == system.MacOS && a.Screens[0].Name == MacOsBuiltinMonitor {
		fss := a.AllScreens[0]
		if fss.Name != MacOsBuiltinMonitor {
			if ScreenDebug {
				log.Printf("ScreenDebug: getScreens: MacOs, builtin is currently primary, but was not originally -- restoring primary monitor as: %s\n", a.Screens[1].Name)
			}
			// assume 2nd one is good..
			a.Screens[0], a.Screens[1] = a.Screens[1], a.Screens[0] // swap
			a.Screens[0].ScreenNumber = 0
			a.Screens[1].ScreenNumber = 1
		}
	}
}

// SaveScreenInfo saves a copy of given screen info to screensAll list if unique
// based on name. Returns true if added a new screen.
func (a *App) SaveScreenInfo(sc *system.Screen) bool {
	_, has := a.FindScreenInfo(sc.Name)
	if has {
		return false
	}
	nsc := &system.Screen{}
	*nsc = *sc
	a.AllScreens = append(a.AllScreens, nsc)
	return true
}

// FindScreenInfo finds saved screen info based on name
func (a *App) FindScreenInfo(name string) (*system.Screen, bool) {
	for _, sc := range a.AllScreens {
		if sc.Name == name {
			return sc, true
		}
	}
	return nil, false
}
