// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"encoding/json"
	"errors"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goki.dev/goosi"
)

var (
	// WinGeomMgr is the manager of window geometry preferences
	WinGeomMgr = WinGeomPrefsMgr{}

	// WinGeomTrace logs window geometry saving / loading functions
	WinGeomTrace = false

	WinGeomNoLockErr = errors.New("WinGeom could not lock lock file")
)

// WinGeomPrefs is the data structure for recording the window geometry
// by window name, screen name.
type WinGeomPrefs map[string]map[string]RenderWinGeom

// WinGeomPrefsMgr is the manager of window geometry preferences.
// Records window geometry in a persistent file, used when opening new windows.
type WinGeomPrefsMgr struct {

	// the full set of window geometries
	Geoms WinGeomPrefs `desc:"the full set of window geometries"`

	// temporary cached geometries -- saved to Geoms after SaveDelay
	Cache WinGeomPrefs `desc:"temporary cached geometries -- saved to Geoms after SaveDelay"`

	// base name of the preferences file in GoGi prefs directory
	FileName string `desc:"base name of the preferences file in GoGi prefs directory"`

	// when prefs were last saved -- if we weren't the last to save, then we need to re-open before modifying
	LastSave time.Time `desc:"when prefs were last saved -- if we weren't the last to save, then we need to re-open before modifying"`

	// if true, we are setting geometry so don't save -- caller must call SettingStart() SettingEnd() to block
	SettingNoSave bool `desc:"if true, we are setting geometry so don't save -- caller must call SettingStart() SettingEnd() to block"`

	// read-write mutex that protects updating of WinGeomPrefs
	Mu sync.RWMutex `desc:"read-write mutex that protects updating of WinGeomPrefs"`

	// wait time before trying to lock file again
	LockSleep time.Duration `desc:"wait time before trying to lock file again"`

	// wait time before saving the Cache into Geoms
	SaveDelay time.Duration `desc:"wait time before saving the Cache into Geoms"`

	// timer for delayed save
	saveTimer *time.Timer `desc:"timer for delayed save"`
}

// Init does initialization if not yet initialized
func (mgr *WinGeomPrefsMgr) Init() {
	if mgr.Geoms == nil {
		mgr.Geoms = make(WinGeomPrefs, 1000)
		mgr.ResetCache()
		mgr.FileName = "win_geom_prefs"
		mgr.LockSleep = 100 * time.Millisecond
		mgr.SaveDelay = 1 * time.Second
	}
}

// ResetCache resets the cache -- call under mutex
func (mgr *WinGeomPrefsMgr) ResetCache() {
	mgr.Cache = make(WinGeomPrefs)
}

// LockFile attempts to create the win_geom_prefs lock file
func (mgr *WinGeomPrefsMgr) LockFile() error {
	pdir := goosi.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, mgr.FileName+".lck")
	for rep := 0; rep < 10; rep++ {
		if _, err := os.Stat(pnm); os.IsNotExist(err) {
			b, _ := time.Now().MarshalJSON()
			err = ioutil.WriteFile(pnm, b, 0644)
			if err == nil {
				return nil
			}
		}
		b, err := ioutil.ReadFile(pnm)
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
		if time.Now().Sub(lts) > 1*time.Second {
			// log.Printf("WinGeomPrefs: lock file stale: %v\n", lts.String())
			os.Remove(pnm)
			continue
		}
		// log.Printf("WinGeomPrefs: waiting for lock file: %v\n", lts.String())
		time.Sleep(mgr.LockSleep)
	}
	// log.Printf("WinGeomPrefs: failed to lock file: %v\n", pnm)
	return WinGeomNoLockErr
}

// UnLockFile unlocks the win_geom_prefs lock file (just removes it)
func (mgr *WinGeomPrefsMgr) UnlockFile() {
	pdir := goosi.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, mgr.FileName+".lck")
	os.Remove(pnm)
}

// NeedToReload returns true if the last save time of prefs file is more recent than
// when we last saved.  Called under mutex.
func (mgr *WinGeomPrefsMgr) NeedToReload() bool {
	pdir := goosi.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, mgr.FileName+".lst")
	if _, err := os.Stat(pnm); os.IsNotExist(err) {
		return false
	}
	var lts time.Time
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		return false
	}
	err = lts.UnmarshalJSON(b)
	if err != nil {
		return false
	}
	eq := lts.Equal(mgr.LastSave)
	if !eq {
		// fmt.Printf("prefs file saved more recently: %v than our last save: %v\n", lts.String(),
		// 	mgr.LastSave.String())
		mgr.LastSave = lts
	}
	return !eq
}

// SaveLastSave saves timestamp (now) of last save to win geom
func (mgr *WinGeomPrefsMgr) SaveLastSave() {
	pdir := goosi.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, mgr.FileName+".lst")
	mgr.LastSave = time.Now()
	b, _ := mgr.LastSave.MarshalJSON()
	ioutil.WriteFile(pnm, b, 0644)
}

// Open RenderWin Geom preferences from GoGi standard prefs directory
// called under mutex or at start
func (mgr *WinGeomPrefsMgr) Open() error {
	mgr.Init()
	pdir := goosi.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, mgr.FileName+".json")
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		// log.Println(err)
		return err
	}
	err = json.Unmarshal(b, &mgr.Geoms)
	if err != nil {
		log.Println(err)
	}
	oldFmt := false
	for _, wps := range mgr.Geoms {
		for _, wp := range wps {
			if wp.DPI == 0 && wp.DPR == 0 {
				oldFmt = true
			}
			break
		}
		break
	}
	if oldFmt {
		log.Printf("WinGeomPrefs: resetting prefs for new format\n")
		mgr.Geoms = make(WinGeomPrefs, 1000)
		mgr.Save() // overwrite
	}
	return err
}

// Save RenderWin Geom Preferences to GoGi standard prefs directory
// assumed to be under mutex and lock still
func (mgr *WinGeomPrefsMgr) Save() error {
	if mgr.Geoms == nil {
		return nil
	}
	pdir := goosi.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, mgr.FileName+".json")
	b, err := json.MarshalIndent(mgr.Geoms, "", "\t")
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(pnm, b, 0644)
	if err != nil {
		log.Println(err)
	} else {
		mgr.SaveLastSave()
	}
	return err
}

// WinName returns window name before first colon, if exists.
// This is the part of the name used to record preferences
func (mgr *WinGeomPrefsMgr) WinName(winName string) string {
	if ci := strings.Index(winName, ":"); ci > 0 {
		return winName[:ci]
	}
	return winName
}

// SettingStart turns on SettingNoSave to prevent subsequent redundant calls to
// save a geometry that was being set from already-saved preferences.
// Must call SettingEnd to turn off (safe to call even if Start not called).
func (mgr *WinGeomPrefsMgr) SettingStart() {
	mgr.Mu.Lock()
	mgr.SettingNoSave = true
	mgr.Mu.Unlock()
}

// SettingEnd turns off SettingNoSave -- safe to call even if Start not called.
func (mgr *WinGeomPrefsMgr) SettingEnd() {
	mgr.Mu.Lock()
	mgr.SettingNoSave = false
	mgr.Mu.Unlock()
}

// RecordPref records current state of window as preference
func (mgr *WinGeomPrefsMgr) RecordPref(win *RenderWin) {
	if !win.IsVisible() {
		return
	}
	wsz := win.RenderWin.Size()
	if wsz == (image.Point{}) {
		if WinGeomTrace {
			log.Printf("WinGeomPrefs: RecordPref: NOT storing null size for win: %v\n", win.Nm)
		}
		return
	}
	pos := win.RenderWin.Position()
	if pos.X == -32000 || pos.Y == -32000 { // windows badness
		if WinGeomTrace {
			log.Printf("WinGeomPrefs: RecordPref: NOT storing very negative pos: %v for win: %v\n", pos, win.Nm)
		}
		return
	}
	mgr.Mu.Lock()
	if mgr.SettingNoSave {
		if WinGeomTrace {
			log.Printf("WinGeomPrefs: RecordPref: SettingNoSave so NOT storing for win: %v\n", win.Nm)
		}
		mgr.Mu.Unlock()
		return
	}
	mgr.Init()

	winName := mgr.WinName(win.Nm)
	sc := win.RenderWin.Screen()
	wgr := RenderWinGeom{DPI: win.LogicalDPI(), DPR: sc.DevicePixelRatio}
	wgr.SetPos(pos)
	wgr.SetSize(wsz)

	if mgr.Cache[winName] == nil {
		mgr.Cache[winName] = make(map[string]RenderWinGeom)
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

// AbortSave cancels any pending saving of the currently-cached info.
// this is called if a screen event occured
func (mgr *WinGeomPrefsMgr) AbortSave() {
	mgr.Mu.Lock()
	defer mgr.Mu.Unlock()
	if mgr.saveTimer != nil {
		mgr.saveTimer.Stop()
		mgr.saveTimer = nil
		if WinGeomTrace {
			if len(mgr.Cache) == 0 {
				log.Printf("WinGeomPrefs: AbortSave: no cached geoms but timer was != nil -- probably already saved\n")
			} else {
				log.Printf("WinGeomPrefs: AbortSave: there are cached geoms -- aborted in time!\n")
			}
		}
	} else {
		if WinGeomTrace {
			log.Printf("WinGeomPrefs: AbortSave: no saveTimer -- already happened or nothing to save\n")
		}
	}
	mgr.ResetCache()
}

// SaveCached saves the cached prefs -- called after timer delay,
// under the Mu.Lock
func (mgr *WinGeomPrefsMgr) SaveCached() {
	mgr.LockFile() // not going to change our behavior if we can't lock!
	if mgr.NeedToReload() {
		mgr.Open()
	}
	for winName, scmap := range mgr.Cache {
		for scName, wgr := range scmap {
			sc := goosi.TheApp.ScreenByName(scName)
			if sc == nil {
				continue
			}
			if mgr.Geoms[winName] == nil {
				mgr.Geoms[winName] = make(map[string]RenderWinGeom)
			}
			mgr.Geoms[winName][sc.Name] = wgr
			if WinGeomTrace {
				log.Printf("WinGeomPrefs: RecordPref: Saving for window: %v pos: %v size: %v  screen: %v  dpi: %v  device pixel ratio: %v\n", winName, wgr.Pos(), wgr.Size(), sc.Name, sc.LogicalDPI, sc.DevicePixelRatio)
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
func (mgr *WinGeomPrefsMgr) Pref(winName string, scrn *goosi.Screen) *RenderWinGeom {
	mgr.Mu.RLock()
	defer mgr.Mu.RUnlock()

	if mgr.Geoms == nil {
		return nil
	}
	winName = mgr.WinName(winName)
	wps, ok := mgr.Cache[winName]
	if !ok {
		wps, ok = mgr.Geoms[winName]
		if !ok {
			return nil
		}
	}

	if scrn == nil {
		scrn = goosi.TheApp.Screen(0)
		if WinGeomTrace {
			log.Printf("WinGeomPrefs: Pref: scrn is nil, using scrn 0: %v\n", scrn.Name)
		}
	}
	wp, ok := wps[scrn.Name]
	if ok {
		wp.ConstrainGeom(scrn)
		if WinGeomTrace {
			log.Printf("WinGeomPrefs: Pref: Setting geom for window: %v pos: %v size: %v  screen: %v  dpi: %v  device pixel ratio: %v\n", winName, wp.Pos(), wp.Size(), scrn.Name, scrn.LogicalDPI, scrn.DevicePixelRatio)
		}
		return &wp
	}
	return nil
}

// DeleteAll deletes the file that saves the position and size of each window,
// by screen, and clear current in-memory cache.  You shouldn't need to use
// this but sometimes useful for testing.
func (mgr *WinGeomPrefsMgr) DeleteAll() {
	mgr.Mu.Lock()
	defer mgr.Mu.Unlock()

	pdir := goosi.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, mgr.FileName+".json")
	os.Remove(pnm)
	mgr.Geoms = make(WinGeomPrefs, 1000)
}

// RestoreAll restores size and position of all windows, for current screen.
// Called when screen changes.
func (mgr *WinGeomPrefsMgr) RestoreAll() {
	RenderWinGlobalMu.Lock()
	defer RenderWinGlobalMu.Unlock()
	if WinGeomTrace {
		log.Printf("WinGeomPrefs: RestoreAll: starting\n")
	}
	mgr.SettingStart()
	for _, w := range AllRenderWins {
		wgp := mgr.Pref(w.Name(), w.RenderWin.Screen())
		if wgp != nil {
			if WinGeomTrace {
				log.Printf("WinGeomPrefs: RestoreAll: restoring geom for window: %v pos: %v size: %v\n", w.Name(), wgp.Pos(), wgp.Size())
			}
			w.RenderWin.SetGeom(wgp.Pos(), wgp.Size())
		}
	}
	mgr.SettingEnd()
	if WinGeomTrace {
		log.Printf("WinGeomPrefs: RestoreAll: done\n")
	}
}

/////////////////////////////////////////////////////////////////////
// RenderWinGeom

// RenderWinGeom records the geometry settings used for a given window
type RenderWinGeom struct {
	DPI float32
	DPR float32
	SX  int
	SY  int
	PX  int
	PY  int
}

func (wg *RenderWinGeom) Size() image.Point {
	return image.Point{wg.SX, wg.SY}
}

func (wg *RenderWinGeom) SetSize(sz image.Point) {
	wg.SX = sz.X
	wg.SY = sz.Y
}

func (wg *RenderWinGeom) Pos() image.Point {
	return image.Point{wg.PX, wg.PY}
}

func (wg *RenderWinGeom) SetPos(ps image.Point) {
	wg.PX = ps.X
	wg.PY = ps.Y
}

// ConstrainGeom constrains geometry based on screen params
func (wg *RenderWinGeom) ConstrainGeom(sc *goosi.Screen) {
	sz, pos := sc.ConstrainWinGeom(image.Point{wg.SX, wg.SY}, image.Point{wg.PX, wg.PY})
	wg.SX = sz.X
	wg.SY = sz.Y
	wg.PX = pos.X
	wg.PY = pos.Y
}
