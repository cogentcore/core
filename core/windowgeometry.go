// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"encoding/json"
	"image"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/system"
)

// theWindowGeometrySaver is the manager of window geometry settings
var theWindowGeometrySaver = windowGeometrySaver{}

// windowGeometries is the data structure for recording the window
// geometries by window name and screen name.
type windowGeometries map[string]map[string]windowGeometry

// windowGeometrySaver records window geometries in a persistent file,
// which is then used when opening new windows to restore.
type windowGeometrySaver struct {

	// the full set of window geometries
	geometries windowGeometries

	// temporary cached geometries -- saved to Geometries after SaveDelay
	cache windowGeometries

	// base name of the settings file in Cogent Core settings directory
	filename string

	// when settings were last saved -- if we weren't the last to save, then we need to re-open before modifying
	lastSave time.Time

	// if true, we are setting geometry so don't save -- caller must call SettingStart() SettingEnd() to block
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
		ws.geometries = make(windowGeometries, 1000)
		ws.resetCache()
		ws.filename = "window-geometry"
		ws.lockSleep = 100 * time.Millisecond
		ws.saveDelay = 1 * time.Second
	}
}

// resetCache resets the cache; call under mutex
func (ws *windowGeometrySaver) resetCache() {
	ws.cache = make(windowGeometries)
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
	err = json.Unmarshal(b, &ws.geometries)
	if err != nil {
		return errors.Log(err)
	}
	oldFmt := false
	for _, wps := range ws.geometries {
		for _, wp := range wps {
			if wp.DPI == 0 && wp.DPR == 0 {
				oldFmt = true
			}
			break
		}
		break
	}
	if oldFmt {
		log.Printf("WindowGeometry: resetting prefs for new format\n")
		ws.geometries = make(windowGeometries, 1000)
		ws.save() // overwrite
	}
	return err
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
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	err = os.WriteFile(pnm, b, 0644)
	if err != nil {
		slog.Error(err.Error())
	} else {
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
	ws.settingNoSave = true
	ws.mu.Unlock()
}

// settingEnd turns off SettingNoSave -- safe to call even if Start not called.
func (ws *windowGeometrySaver) settingEnd() {
	ws.mu.Lock()
	ws.settingNoSave = false
	ws.mu.Unlock()
}

// recordPref records current state of window as preference
func (ws *windowGeometrySaver) recordPref(win *renderWindow) {
	if !win.isVisible() {
		return
	}
	win.SystemWindow.Lock()
	wsz := win.SystemWindow.Size()
	win.SystemWindow.Unlock()
	if wsz == (image.Point{}) {
		if DebugSettings.WinGeomTrace {
			log.Printf("WindowGeometry: RecordPref: NOT storing null size for win: %v\n", win.name)
		}
		return
	}
	pos := win.SystemWindow.Position()
	if pos.X == -32000 || pos.Y == -32000 { // windows badness
		if DebugSettings.WinGeomTrace {
			log.Printf("WindowGeometry: RecordPref: NOT storing very negative pos: %v for win: %v\n", pos, win.name)
		}
		return
	}
	ws.mu.Lock()
	if ws.settingNoSave {
		if DebugSettings.WinGeomTrace {
			log.Printf("WindowGeometry: RecordPref: SettingNoSave so NOT storing for win: %v\n", win.name)
		}
		ws.mu.Unlock()
		return
	}
	ws.init()

	winName := ws.windowName(win.title)
	sc := win.SystemWindow.Screen()
	wgr := windowGeometry{DPI: win.logicalDPI(), DPR: sc.DevicePixelRatio, Fullscreen: win.SystemWindow.Is(system.Fullscreen)}
	wgr.setPos(pos)
	wgr.setSize(wsz)

	if ws.cache[winName] == nil {
		ws.cache[winName] = make(map[string]windowGeometry)
	}
	ws.cache[winName][sc.Name] = wgr
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
	for winName, scmap := range ws.cache {
		for scName, wgr := range scmap {
			sc := system.TheApp.ScreenByName(scName)
			if sc == nil {
				continue
			}
			if ws.geometries[winName] == nil {
				ws.geometries[winName] = make(map[string]windowGeometry)
			}
			ws.geometries[winName][sc.Name] = wgr
			if DebugSettings.WinGeomTrace {
				log.Printf("WindowGeometry: RecordPref: Saving for window: %v pos: %v size: %v  screen: %v  dpi: %v  device pixel ratio: %v\n", winName, wgr.pos(), wgr.size(), sc.Name, sc.LogicalDPI, sc.DevicePixelRatio)
			}
		}
	}
	ws.resetCache()
	ws.save()
	ws.unlockFile()
}

// pref returns an existing preference for given window name, for given screen.
// if the window name has a colon, only the part prior to the colon is used.
// if no saved pref is available for that screen, nil is returned.
func (ws *windowGeometrySaver) pref(winName string, scrn *system.Screen) *windowGeometry {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	if ws.geometries == nil {
		return nil
	}
	winName = ws.windowName(winName)
	wps, ok := ws.cache[winName]
	if !ok {
		wps, ok = ws.geometries[winName]
		if !ok {
			return nil
		}
	}

	if scrn == nil {
		scrn = system.TheApp.Screen(0)
		if DebugSettings.WinGeomTrace {
			log.Printf("WindowGeometry: Pref: scrn is nil, using scrn 0: %v\n", scrn.Name)
		}
	}
	wp, ok := wps[scrn.Name]
	if ok {
		wp.constrainGeom(scrn)
		if DebugSettings.WinGeomTrace {
			log.Printf("WindowGeometry: Pref: Setting geom for window: %v pos: %v size: %v  screen: %v  dpi: %v  device pixel ratio: %v\n", winName, wp.pos(), wp.size(), scrn.Name, scrn.LogicalDPI, scrn.DevicePixelRatio)
		}
		return &wp
	}
	return nil
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
	ws.geometries = make(windowGeometries, 1000)
}

// restoreAll restores size and position of all windows, for current screen.
// Called when screen changes.
func (ws *windowGeometrySaver) restoreAll() {
	renderWindowGlobalMu.Lock()
	defer renderWindowGlobalMu.Unlock()
	if DebugSettings.WinGeomTrace {
		log.Printf("WindowGeometry: RestoreAll: starting\n")
	}
	ws.settingStart()
	for _, w := range AllRenderWindows {
		wgp := ws.pref(w.title, w.SystemWindow.Screen())
		if wgp != nil {
			if DebugSettings.WinGeomTrace {
				log.Printf("WindowGeometry: RestoreAll: restoring geom for window: %v pos: %v size: %v\n", w.name, wgp.pos(), wgp.size())
			}
			w.SystemWindow.SetGeom(wgp.pos(), wgp.size())
		}
	}
	ws.settingEnd()
	if DebugSettings.WinGeomTrace {
		log.Printf("WindowGeometry: RestoreAll: done\n")
	}
}

// windowGeometry records the geometry settings used for a given window
type windowGeometry struct {
	DPI        float32
	DPR        float32
	SX         int
	SY         int
	PX         int
	PY         int
	Fullscreen bool
}

func (wg *windowGeometry) size() image.Point {
	return image.Point{wg.SX, wg.SY}
}

func (wg *windowGeometry) setSize(sz image.Point) {
	wg.SX = sz.X
	wg.SY = sz.Y
}

func (wg *windowGeometry) pos() image.Point {
	return image.Point{wg.PX, wg.PY}
}

func (wg *windowGeometry) setPos(ps image.Point) {
	wg.PX = ps.X
	wg.PY = ps.Y
}

// constrainGeom constrains geometry based on screen params
func (wg *windowGeometry) constrainGeom(sc *system.Screen) {
	sz, pos := sc.ConstrainWinGeom(image.Point{wg.SX, wg.SY}, image.Point{wg.PX, wg.PY})
	wg.SX = sz.X
	wg.SY = sz.Y
	wg.PX = pos.X
	wg.PY = pos.Y
}
