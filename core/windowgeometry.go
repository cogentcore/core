// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"encoding/json"
	"errors"
	"image"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/system"
)

// TheWindowGeometrySaver is the manager of window geometry settings
var TheWindowGeometrySaver = WindowGeometrySaver{}

// WindowGeometries is the data structure for recording the window
// geometries by window name and screen name.
type WindowGeometries map[string]map[string]WindowGeometry

// WindowGeometrySaver records window geometries in a persistent file,
// which is then used when opening new windows to restore.
type WindowGeometrySaver struct {

	// the full set of window geometries
	Geometries WindowGeometries

	// temporary cached geometries -- saved to Geometries after SaveDelay
	Cache WindowGeometries

	// base name of the settings file in Cogent Core settings directory
	Filename string

	// when settings were last saved -- if we weren't the last to save, then we need to re-open before modifying
	LastSave time.Time

	// if true, we are setting geometry so don't save -- caller must call SettingStart() SettingEnd() to block
	SettingNoSave bool

	// read-write mutex that protects updating of WindowGeometry
	Mu sync.RWMutex

	// wait time before trying to lock file again
	LockSleep time.Duration

	// wait time before saving the Cache into Geometries
	SaveDelay time.Duration

	// timer for delayed save
	saveTimer *time.Timer
}

// Init does initialization if not yet initialized
func (mgr *WindowGeometrySaver) Init() {
	if mgr.Geometries == nil {
		mgr.Geometries = make(WindowGeometries, 1000)
		mgr.ResetCache()
		mgr.Filename = "window-geometry"
		mgr.LockSleep = 100 * time.Millisecond
		mgr.SaveDelay = 1 * time.Second
	}
}

// ResetCache resets the cache -- call under mutex
func (mgr *WindowGeometrySaver) ResetCache() {
	mgr.Cache = make(WindowGeometries)
}

// LockFile attempts to create the window geometry lock file
func (mgr *WindowGeometrySaver) LockFile() error {
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, mgr.Filename+".lck")
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
			time.Sleep(mgr.LockSleep)
			continue
		}
		var lts time.Time
		err = lts.UnmarshalJSON(b)
		if err != nil {
			time.Sleep(mgr.LockSleep)
			continue
		}
		if time.Since(lts) > 1*time.Second {
			// log.Printf("WindowGeometry: lock file stale: %v\n", lts.String())
			os.Remove(pnm)
			continue
		}
		// log.Printf("WindowGeometry: waiting for lock file: %v\n", lts.String())
		time.Sleep(mgr.LockSleep)
	}
	return errors.New("WinGeom could not lock lock file")
}

// UnLockFile unlocks the window geometry lock file (just removes it)
func (mgr *WindowGeometrySaver) UnlockFile() {
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, mgr.Filename+".lck")
	os.Remove(pnm)
}

// NeedToReload returns true if the last save time of settings file is more recent than
// when we last saved.  Called under mutex.
func (mgr *WindowGeometrySaver) NeedToReload() bool {
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, mgr.Filename+".lst")
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
	eq := lts.Equal(mgr.LastSave)
	if !eq {
		// fmt.Printf("settings file saved more recently: %v than our last save: %v\n", lts.String(),
		// 	mgr.LastSave.String())
		mgr.LastSave = lts
	}
	return !eq
}

// SaveLastSave saves timestamp (now) of last save to win geom
func (mgr *WindowGeometrySaver) SaveLastSave() {
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, mgr.Filename+".lst")
	mgr.LastSave = time.Now()
	b, _ := mgr.LastSave.MarshalJSON()
	os.WriteFile(pnm, b, 0644)
}

// Open RenderWindow Geom settings from Cogent Core standard settings directory
// called under mutex or at start
func (mgr *WindowGeometrySaver) Open() error {
	mgr.Init()
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, mgr.Filename+".json")
	b, err := os.ReadFile(pnm)
	if err != nil {
		// slog.Error(err.Error())rror())
		return err
	}
	err = json.Unmarshal(b, &mgr.Geometries)
	if err != nil {
		slog.Error(err.Error())
	}
	oldFmt := false
	for _, wps := range mgr.Geometries {
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
		mgr.Geometries = make(WindowGeometries, 1000)
		mgr.Save() // overwrite
	}
	return err
}

// Save RenderWindow Geom Settings to Cogent Core standard prefs directory
// assumed to be under mutex and lock still
func (mgr *WindowGeometrySaver) Save() error {
	if mgr.Geometries == nil {
		return nil
	}
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, mgr.Filename+".json")
	b, err := json.MarshalIndent(mgr.Geometries, "", "\t")
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	err = os.WriteFile(pnm, b, 0644)
	if err != nil {
		slog.Error(err.Error())
	} else {
		mgr.SaveLastSave()
	}
	return err
}

// WinName returns window name before first colon, if exists.
// This is the part of the name used to record settings
func (mgr *WindowGeometrySaver) WinName(winName string) string {
	if ci := strings.Index(winName, ":"); ci > 0 {
		return winName[:ci]
	}
	return winName
}

// SettingStart turns on SettingNoSave to prevent subsequent redundant calls to
// save a geometry that was being set from already-saved settings.
// Must call SettingEnd to turn off (safe to call even if Start not called).
func (mgr *WindowGeometrySaver) SettingStart() {
	mgr.Mu.Lock()
	mgr.SettingNoSave = true
	mgr.Mu.Unlock()
}

// SettingEnd turns off SettingNoSave -- safe to call even if Start not called.
func (mgr *WindowGeometrySaver) SettingEnd() {
	mgr.Mu.Lock()
	mgr.SettingNoSave = false
	mgr.Mu.Unlock()
}

// RecordPref records current state of window as preference
func (mgr *WindowGeometrySaver) RecordPref(win *RenderWindow) {
	if !win.IsVisible() {
		return
	}
	wsz := win.SystemWindow.Size()
	if wsz == (image.Point{}) {
		if DebugSettings.WinGeomTrace {
			log.Printf("WindowGeometry: RecordPref: NOT storing null size for win: %v\n", win.Name)
		}
		return
	}
	pos := win.SystemWindow.Position()
	if pos.X == -32000 || pos.Y == -32000 { // windows badness
		if DebugSettings.WinGeomTrace {
			log.Printf("WindowGeometry: RecordPref: NOT storing very negative pos: %v for win: %v\n", pos, win.Name)
		}
		return
	}
	mgr.Mu.Lock()
	if mgr.SettingNoSave {
		if DebugSettings.WinGeomTrace {
			log.Printf("WindowGeometry: RecordPref: SettingNoSave so NOT storing for win: %v\n", win.Name)
		}
		mgr.Mu.Unlock()
		return
	}
	mgr.Init()

	winName := mgr.WinName(win.Title)
	sc := win.SystemWindow.Screen()
	wgr := WindowGeometry{DPI: win.LogicalDPI(), DPR: sc.DevicePixelRatio, Fullscreen: win.SystemWindow.Is(system.Fullscreen)}
	wgr.SetPos(pos)
	wgr.SetSize(wsz)

	if mgr.Cache[winName] == nil {
		mgr.Cache[winName] = make(map[string]WindowGeometry)
	}
	mgr.Cache[winName][sc.Name] = wgr
	if mgr.saveTimer == nil {
		mgr.saveTimer = time.AfterFunc(time.Duration(mgr.SaveDelay), func() {
			mgr.Mu.Lock()
			mgr.SaveCached()
			mgr.saveTimer = nil
			mgr.Mu.Unlock()
		})
	}
	mgr.Mu.Unlock()
}

// AbortSave cancels any pending saving of the currently cached info.
// this is called if a screen event occured
func (mgr *WindowGeometrySaver) AbortSave() {
	mgr.Mu.Lock()
	defer mgr.Mu.Unlock()
	if mgr.saveTimer != nil {
		mgr.saveTimer.Stop()
		mgr.saveTimer = nil
		if DebugSettings.WinGeomTrace {
			if len(mgr.Cache) == 0 {
				log.Printf("WindowGeometry: AbortSave: no cached geometries but timer was != nil -- probably already saved\n")
			} else {
				log.Printf("WindowGeometry: AbortSave: there are cached geometries -- aborted in time!\n")
			}
		}
	} else {
		if DebugSettings.WinGeomTrace {
			log.Printf("WindowGeometry: AbortSave: no saveTimer -- already happened or nothing to save\n")
		}
	}
	mgr.ResetCache()
}

// SaveCached saves the cached prefs -- called after timer delay,
// under the Mu.Lock
func (mgr *WindowGeometrySaver) SaveCached() {
	mgr.LockFile() // not going to change our behavior if we can't lock!
	if mgr.NeedToReload() {
		mgr.Open()
	}
	for winName, scmap := range mgr.Cache {
		for scName, wgr := range scmap {
			sc := system.TheApp.ScreenByName(scName)
			if sc == nil {
				continue
			}
			if mgr.Geometries[winName] == nil {
				mgr.Geometries[winName] = make(map[string]WindowGeometry)
			}
			mgr.Geometries[winName][sc.Name] = wgr
			if DebugSettings.WinGeomTrace {
				log.Printf("WindowGeometry: RecordPref: Saving for window: %v pos: %v size: %v  screen: %v  dpi: %v  device pixel ratio: %v\n", winName, wgr.Pos(), wgr.Size(), sc.Name, sc.LogicalDPI, sc.DevicePixelRatio)
			}
		}
	}
	mgr.ResetCache()
	mgr.Save()
	mgr.UnlockFile()
}

// Pref returns an existing preference for given window name, for given screen.
// if the window name has a colon, only the part prior to the colon is used.
// if no saved pref is available for that screen, nil is returned.
func (mgr *WindowGeometrySaver) Pref(winName string, scrn *system.Screen) *WindowGeometry {
	mgr.Mu.RLock()
	defer mgr.Mu.RUnlock()

	if mgr.Geometries == nil {
		return nil
	}
	winName = mgr.WinName(winName)
	wps, ok := mgr.Cache[winName]
	if !ok {
		wps, ok = mgr.Geometries[winName]
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
		wp.ConstrainGeom(scrn)
		if DebugSettings.WinGeomTrace {
			log.Printf("WindowGeometry: Pref: Setting geom for window: %v pos: %v size: %v  screen: %v  dpi: %v  device pixel ratio: %v\n", winName, wp.Pos(), wp.Size(), scrn.Name, scrn.LogicalDPI, scrn.DevicePixelRatio)
		}
		return &wp
	}
	return nil
}

// DeleteAll deletes the file that saves the position and size of each window,
// by screen, and clear current in-memory cache.  You shouldn't need to use
// this but sometimes useful for testing.
func (mgr *WindowGeometrySaver) DeleteAll() {
	mgr.Mu.Lock()
	defer mgr.Mu.Unlock()

	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, mgr.Filename+".json")
	os.Remove(pnm)
	mgr.Geometries = make(WindowGeometries, 1000)
}

// RestoreAll restores size and position of all windows, for current screen.
// Called when screen changes.
func (mgr *WindowGeometrySaver) RestoreAll() {
	RenderWindowGlobalMu.Lock()
	defer RenderWindowGlobalMu.Unlock()
	if DebugSettings.WinGeomTrace {
		log.Printf("WindowGeometry: RestoreAll: starting\n")
	}
	mgr.SettingStart()
	for _, w := range AllRenderWindows {
		wgp := mgr.Pref(w.Title, w.SystemWindow.Screen())
		if wgp != nil {
			if DebugSettings.WinGeomTrace {
				log.Printf("WindowGeometry: RestoreAll: restoring geom for window: %v pos: %v size: %v\n", w.Name, wgp.Pos(), wgp.Size())
			}
			w.SystemWindow.SetGeom(wgp.Pos(), wgp.Size())
		}
	}
	mgr.SettingEnd()
	if DebugSettings.WinGeomTrace {
		log.Printf("WindowGeometry: RestoreAll: done\n")
	}
}

// WindowGeometry records the geometry settings used for a given window
type WindowGeometry struct {
	DPI        float32
	DPR        float32
	SX         int
	SY         int
	PX         int
	PY         int
	Fullscreen bool
}

func (wg *WindowGeometry) Size() image.Point {
	return image.Point{wg.SX, wg.SY}
}

func (wg *WindowGeometry) SetSize(sz image.Point) {
	wg.SX = sz.X
	wg.SY = sz.Y
}

func (wg *WindowGeometry) Pos() image.Point {
	return image.Point{wg.PX, wg.PY}
}

func (wg *WindowGeometry) SetPos(ps image.Point) {
	wg.PX = ps.X
	wg.PY = ps.Y
}

// ConstrainGeom constrains geometry based on screen params
func (wg *WindowGeometry) ConstrainGeom(sc *system.Screen) {
	sz, pos := sc.ConstrainWinGeom(image.Point{wg.SX, wg.SY}, image.Point{wg.PX, wg.PY})
	wg.SX = sz.X
	wg.SY = sz.Y
	wg.PX = pos.X
	wg.PY = pos.Y
}
