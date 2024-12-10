// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"encoding/json"
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

	// the full set of window geometries
	geometries screenConfigGeometries

	// temporary cached geometries: saved to geometries after SaveDelay
	cache screenConfigGeometries

	// base name of the settings file in Cogent Core settings directory
	filename string

	// when settings were last saved: if we weren't the last to save,
	// then we need to re-open before modifying.
	lastSave time.Time

	// if true, we are setting geometry so don't save;
	// Caller must call SettingStart() SettingEnd() to block.
	settingNoSave bool

	// read-write mutex that protects updating of WindowGeometry
	mu sync.RWMutex

	// wait time before trying to lock file again
	lockSleep time.Duration

	// wait time before saving the Cache into Geometries
	saveDelay time.Duration

	// timer for delayed save
	saveTimer *time.Timer
}

// init does initialization if not yet initialized
func (ws *windowGeometrySaver) init() {
	if ws.geometries == nil {
		ws.geometries = make(screenConfigGeometries)
		ws.resetCache()
		ws.filename = "window-geometry-0.3.6"
		ws.lockSleep = 100 * time.Millisecond
		ws.saveDelay = 1 * time.Second
	}
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

// lockFile attempts to create the window geometry lock file
func (ws *windowGeometrySaver) lockFile() error {
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, ws.filename+".lck")
	for rep := 0; rep < 10; rep++ {
		if _, err := os.Stat(pnm); os.IsNotExist(err) {
			b, _ := time.Now().MarshalJSON()
			err = os.WriteFile(pnm, b, 0644)
			if err == nil {
				return nil
			}
		}
		b, err := os.ReadFile(pnm)
		if err != nil {
			time.Sleep(ws.lockSleep)
			continue
		}
		var lts time.Time
		err = lts.UnmarshalJSON(b)
		if err != nil {
			time.Sleep(ws.lockSleep)
			continue
		}
		if time.Since(lts) > 1*time.Second {
			// log.Printf("WindowGeometry: lock file stale: %v\n", lts.String())
			os.Remove(pnm)
			continue
		}
		// log.Printf("WindowGeometry: waiting for lock file: %v\n", lts.String())
		time.Sleep(ws.lockSleep)
	}
	return errors.New("WinGeom could not lock lock file")
}

// UnLockFile unlocks the window geometry lock file (just removes it)
func (ws *windowGeometrySaver) unlockFile() {
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, ws.filename+".lck")
	os.Remove(pnm)
}

// needToReload returns true if the last save time of settings file is more recent than
// when we last saved.  Called under mutex.
func (ws *windowGeometrySaver) needToReload() bool {
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, ws.filename+".lst")
	if _, err := os.Stat(pnm); os.IsNotExist(err) {
		return false
	}
	var lts time.Time
	b, err := os.ReadFile(pnm)
	if err != nil {
		return false
	}
	err = lts.UnmarshalJSON(b)
	if err != nil {
		return false
	}
	eq := lts.Equal(ws.lastSave)
	if !eq {
		// fmt.Printf("settings file saved more recently: %v than our last save: %v\n", lts.String(),
		// 	mgr.LastSave.String())
		ws.lastSave = lts
	}
	return !eq
}

// saveLastSave saves timestamp (now) of last save to win geom
func (ws *windowGeometrySaver) saveLastSave() {
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, ws.filename+".lst")
	ws.lastSave = time.Now()
	b, _ := ws.lastSave.MarshalJSON()
	os.WriteFile(pnm, b, 0644)
}

// open RenderWindow Geom settings from Cogent Core standard settings directory
// called under mutex or at start
func (ws *windowGeometrySaver) open() error {
	ws.init()
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, ws.filename+".json")
	b, err := os.ReadFile(pnm)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &ws.geometries)
}

// save RenderWindow Geom Settings to Cogent Core standard prefs directory
// assumed to be under mutex and lock still
func (ws *windowGeometrySaver) save() error {
	if ws.geometries == nil {
		return nil
	}
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, ws.filename+".json")
	b, err := json.Marshal(ws.geometries)
	if errors.Log(err) != nil {
		return err
	}
	err = os.WriteFile(pnm, b, 0644)
	if errors.Log(err) == nil {
		ws.saveLastSave()
	}
	return err
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
	sgs := ws.geometries[cfg]
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
// under the Mu.Lock
func (ws *windowGeometrySaver) saveCached() {
	ws.lockFile() // not going to change our behavior if we can't lock!
	if ws.needToReload() {
		ws.open()
	}
	if DebugSettings.WindowGeometryTrace {
		log.Println("WindowGeometry: saveCached")
	}
	for cfg, sgsc := range ws.cache {
		for winName, wgs := range sgsc {
			sg := ws.geometries[cfg]
			if sg == nil {
				sg = make(map[string]windowGeometries)
			}
			sg[winName] = wgs
			ws.geometries[cfg] = sg
		}
	}
	ws.resetCache()
	ws.save()
	ws.unlockFile()
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
		sgs, ok = ws.geometries[cfg]
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

	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, ws.filename+".json")
	errors.Log(os.Remove(pnm))
	ws.geometries = make(screenConfigGeometries)
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
