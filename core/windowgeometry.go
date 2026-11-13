// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/sharedfile"
	"cogentcore.org/core/system"
)

var (
	// theWindowGeometrySaver is the manager of window geometry settings
	theWindowGeometrySaver = windowGeometrySaver{}
)

// screenConfigGeometries has the window geometry data for different
// screen configurations, where a screen configuration is a specific
// set of available screens, naturally corresponding to the
// home vs. office vs. travel usage of a laptop, for example,
// with different sets of screens available in each location.
// Each such configuration has a different set of saved window geometries,
// which is restored when the set of screens changes, so your windows will
// be restored to their last positions and sizes for each such configuration.
type screenConfigGeometries map[string]map[string]windowGeometries

// screenConfig returns the current screen configuration string,
// which is the alpha-sorted list of current screen names.
func screenConfig() string {
	ns := TheApp.NScreens()
	if ns == 0 {
		return "none"
	}
	scs := make([]string, ns)
	for i := range ns {
		scs[i] = TheApp.Screen(i).Name
	}
	slices.Sort(scs)
	return strings.Join(scs, "|")
}

// windowGeometrySaver records window geometries in a persistent file,
// which is then used when opening new windows to restore.
type windowGeometrySaver struct {

	// the full set of window geometries, in the file shared by all apps,
	// which is read again when another app has changed it
	geometries sharedfile.File[screenConfigGeometries]

	// temporary cached geometries: saved to geometries after SaveDelay
	cache screenConfigGeometries

	// base name of the settings file in Cogent Core settings directory
	filename string

	// whether init has been done
	inited bool

	// if true, we are setting geometry so don't save;
	// Caller must call SettingStart() SettingEnd() to block.
	settingNoSave bool

	// read-write mutex that protects updating of WindowGeometry
	mu sync.RWMutex

	// wait time before saving the Cache into Geometries
	saveDelay time.Duration

	// timer for delayed save
	saveTimer *time.Timer
}

// init does initialization if not yet initialized
func (ws *windowGeometrySaver) init() {
	if ws.inited {
		return
	}
	ws.inited = true
	ws.resetCache()
	ws.filename = "window-geometry-0.3.6"
	ws.geometries.Path = filepath.Join(TheApp.CogentCoreDataDir(), ws.filename+".json")
	ws.saveDelay = 1 * time.Second
}

// shouldSave returns whether the window geometry should be saved based on
// the platform: only for desktop native platforms.
func (ws *windowGeometrySaver) shouldSave() bool {
	return !TheApp.Platform().IsMobile() && TheApp.Platform() != system.Offscreen && !DebugSettings.DisableWindowGeometrySaver
}

// resetCache resets the cache; call under mutex
func (ws *windowGeometrySaver) resetCache() {
	ws.cache = make(screenConfigGeometries)
}

// open opens the window geometries file from the Cogent Core data directory,
// if it changed since it was last read. Called under mutex or at start.
func (ws *windowGeometrySaver) open() error {
	ws.init()
	return ws.geometries.Reload()
}

// windowName returns window name before first colon, if exists.
// This is the part of the name used to record settings
func (ws *windowGeometrySaver) windowName(winName string) string {
	if ci := strings.Index(winName, ":"); ci > 0 {
		return winName[:ci]
	}
	return winName
}

// settingStart turns on SettingNoSave to prevent subsequent redundant calls to
// save a geometry that was being set from already-saved settings.
// Must call SettingEnd to turn off (safe to call even if Start not called).
func (ws *windowGeometrySaver) settingStart() {
	ws.mu.Lock()
	ws.resetCache() // get rid of anything just saved prior to this -- sus.
	ws.settingNoSave = true
	ws.mu.Unlock()
}

// settingEnd turns off SettingNoSave -- safe to call even if Start not called.
func (ws *windowGeometrySaver) settingEnd() {
	ws.mu.Lock()
	ws.settingNoSave = false
	ws.mu.Unlock()
}

// record records current state of window as preference
func (ws *windowGeometrySaver) record(win *renderWindow) {
	if !ws.shouldSave() || !win.isVisible() || win.SystemWindow.Is(system.Fullscreen) {
		return
	}
	win.SystemWindow.Lock()
	wsz := win.SystemWindow.Size()
	win.SystemWindow.Unlock()
	if wsz == (image.Point{}) {
		if DebugSettings.WindowGeometryTrace {
			log.Printf("WindowGeometry: Record: NOT storing null size for win: %v\n", win.name)
		}
		return
	}
	sc := win.SystemWindow.Screen()
	pos := win.SystemWindow.Position(sc)
	if TheApp.Platform() == system.Windows && pos.X == -32000 || pos.Y == -32000 { // windows badness
		if DebugSettings.WindowGeometryTrace {
			log.Printf("WindowGeometry: Record: NOT storing very negative pos: %v for win: %v\n", pos, win.name)
		}
		return
	}
	ws.mu.Lock()
	if ws.settingNoSave {
		if DebugSettings.WindowGeometryTrace {
			log.Printf("WindowGeometry: Record: SettingNoSave so NOT storing for win: %v\n", win.name)
		}
		ws.mu.Unlock()
		return
	}
	ws.init()

	cfg := screenConfig()
	winName := ws.windowName(win.title)
	wgr := windowGeometry{DPI: win.logicalDPI(), DPR: sc.DevicePixelRatio, Max: win.SystemWindow.Is(system.Maximized)}
	wgr.Pos = pos
	wgr.Size = wsz

	// first get copy of stored data
	sgs := ws.geometries.Data[cfg]
	if sgs == nil {
		sgs = make(map[string]windowGeometries)
	}
	var wgs windowGeometries
	if swgs, ok := sgs[winName]; ok {
		wgs.Last = swgs.Last
		wgs.Screens = maps.Clone(swgs.Screens)
	} else {
		wgs.Screens = make(map[string]windowGeometry)
	}

	// then look for current cache data
	sgsc := ws.cache[cfg]
	if sgsc == nil {
		sgsc = make(map[string]windowGeometries)
	}
	wgsc, hasCache := sgsc[winName]
	if hasCache {
		wgs.Last = wgsc.Last
		for k, v := range wgsc.Screens {
			wgs.Screens[k] = v
		}
	}
	wgs.Screens[sc.Name] = wgr
	wgs.Last = sc.Name
	sgsc[winName] = wgs
	ws.cache[cfg] = sgsc

	if DebugSettings.WindowGeometryTrace {
		log.Printf("WindowGeometry: Record win: %q screen: %q cfg: %q geom: %s", winName, sc.Name, cfg, wgr.String())
	}

	if ws.saveTimer == nil {
		ws.saveTimer = time.AfterFunc(time.Duration(ws.saveDelay), func() {
			ws.mu.Lock()
			ws.saveCached()
			ws.saveTimer = nil
			ws.mu.Unlock()
		})
	}
	ws.mu.Unlock()
}

// saveCached saves the cached prefs -- called after timer delay,
// under the Mu.Lock. The file is locked while it is saved, after reading
// the changes of other apps, which the cached geometries are merged into.
func (ws *windowGeometrySaver) saveCached() {
	if DebugSettings.WindowGeometryTrace {
		log.Println("WindowGeometry: saveCached")
	}
	errors.Log(ws.geometries.Update(func(geoms *screenConfigGeometries) {
		if *geoms == nil {
			*geoms = make(screenConfigGeometries)
		}
		for cfg, sgsc := range ws.cache {
			for winName, wgs := range sgsc {
				sg := (*geoms)[cfg]
				if sg == nil {
					sg = make(map[string]windowGeometries)
				}
				sg[winName] = wgs
				(*geoms)[cfg] = sg
			}
		}
	}))
	ws.resetCache()
}

// get returns saved geometry for given window name, returning
// nil if there is no saved info. The last saved screen is used
// if it is currently available (connected); otherwise the given screen
// name is used if non-empty; otherwise the default screen 0 is used.
// If no saved info is found for any active screen, nil is returned.
// The screen used for the preferences is returned, and should be used
// to set the screen for a new window.
// If the window name has a colon, only the part prior to the colon is used.
func (ws *windowGeometrySaver) get(winName, screenName string) (*windowGeometry, *system.Screen) {
	if !ws.shouldSave() {
		return nil, nil
	}
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	cfg := screenConfig()
	winName = ws.windowName(winName)
	var wgs windowGeometries

	fromMain := false
	sgs := ws.cache[cfg]
	ok := false
	if sgs != nil {
		wgs, ok = sgs[winName]
	}
	if !ok {
		sgs, ok = ws.geometries.Data[cfg]
		if !ok {
			return nil, nil
		}
		wgs, ok = sgs[winName]
		fromMain = true
	}
	if !ok {
		return nil, nil
	}
	wgr, sc := wgs.getForScreen(screenName)
	if wgr != nil {
		wgr.constrainGeom(sc)
		if DebugSettings.WindowGeometryTrace {
			log.Printf("WindowGeometry: Got geom for window: %q screen: %q lastScreen: %q cfg: %q geom: %s fromMain: %v\n", winName, sc.Name, wgs.Last, cfg, wgr.String(), fromMain)
		}
		return wgr, sc
	}
	return nil, nil
}

// deleteAll deletes the file that saves the position and size of each window,
// by screen, and clear current in-memory cache.  You shouldn't need to use
// this but sometimes useful for testing.
func (ws *windowGeometrySaver) deleteAll() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.init()
	errors.Log(os.Remove(ws.geometries.Path))
	ws.geometries.Data = nil
}

// restoreAll restores size and position of all windows, for current screen.
// Called when screen changes.
func (ws *windowGeometrySaver) restoreAll() {
	if !ws.shouldSave() {
		return
	}
	renderWindowGlobalMu.Lock()
	defer renderWindowGlobalMu.Unlock()
	if DebugSettings.WindowGeometryTrace {
		log.Printf("WindowGeometry: RestoreAll: starting\n")
	}
	ws.settingStart()
	for _, w := range AllRenderWindows {
		wgp, sc := ws.get(w.title, "")
		if wgp != nil && !w.SystemWindow.Is(system.Fullscreen) {
			if DebugSettings.WindowGeometryTrace {
				log.Printf("WindowGeometry: RestoreAll: restoring geom for window: %v screen: %s geom: %s\n", w.name, sc.Name, wgp.String())
			}
			w.SystemWindow.SetGeometry(false, wgp.Pos, wgp.Size, sc)
		}
	}
	ws.settingEnd()
	if DebugSettings.WindowGeometryTrace {
		log.Printf("WindowGeometry: RestoreAll: done\n")
	}
}

// windowGeometries holds the window geometries for a given window
// across different screens, and the last screen used.
type windowGeometries struct {
	Last    string                    // Last screen
	Screens map[string]windowGeometry // Screen map
}

// getForScreen returns saved geometry for an active (connected) Screen,
// searching in order of: last screen saved, given screen name, and then
// going through the list of available screens in order.
// returns nil if no saved geometry info is available for any active screen.
func (wgs *windowGeometries) getForScreen(screenName string) (*windowGeometry, *system.Screen) {
	sc := TheApp.ScreenByName(wgs.Last)
	if sc != nil {
		wgr := wgs.Screens[wgs.Last]
		return &wgr, sc
	}
	sc = TheApp.ScreenByName(screenName)
	if sc != nil {
		if wgr, ok := wgs.Screens[screenName]; ok {
			return &wgr, sc
		}
	}
	ns := TheApp.NScreens()
	for i := range ns {
		sc = TheApp.Screen(i)
		if wgr, ok := wgs.Screens[sc.Name]; ok {
			return &wgr, sc
		}
	}
	return nil, nil
}

// windowGeometry records the geometry settings used for
// a certain screen and window pair.
type windowGeometry struct {
	DPI  float32
	DPR  float32 // Device Pixel Ratio
	Size image.Point
	Pos  image.Point
	Max  bool // Maximized
}

func (wg *windowGeometry) String() string {
	return fmt.Sprintf("DPI: %g  DPR: %g  Size: %v  Pos: %v  Max: %v", wg.DPI, wg.DPR, wg.Size, wg.Pos, wg.Max)
}

// constrainGeom constrains geometry based on screen params
func (wg *windowGeometry) constrainGeom(sc *system.Screen) {
	wg.Pos, wg.Size = sc.ConstrainWindowGeometry(wg.Pos, wg.Size)
}
