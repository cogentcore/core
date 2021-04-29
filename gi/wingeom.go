// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ints"
	"github.com/goki/mat32"
)

var WinGeomTrace = false

var WinGeomPrefs = WindowGeomPrefs{}

// WindowGeom records the geometry settings used for a given window
type WindowGeom struct {
	DPI float32
	DPR float32
	SX  int
	SY  int
	PX  int
	PY  int
}

func (wg *WindowGeom) Size() image.Point {
	return image.Point{wg.SX, wg.SY}
}

func (wg *WindowGeom) SetSize(sz image.Point) {
	wg.SX = sz.X
	wg.SY = sz.Y
}

func (wg *WindowGeom) ScaleSize(fact float32) {
	wg.SX = int(float32(wg.SX) * fact)
	wg.SY = int(float32(wg.SY) * fact)
}

func (wg *WindowGeom) Pos() image.Point {
	return image.Point{wg.PX, wg.PY}
}

func (wg *WindowGeom) SetPos(ps image.Point) {
	wg.PX = ps.X
	wg.PY = ps.Y
}

func (wg *WindowGeom) ScalePos(fact float32) {
	wg.PX = int(float32(wg.PX) * fact)
	wg.PY = int(float32(wg.PY) * fact)
}

func (wg *WindowGeom) FitInSize(sz image.Point) {
	wg.SX = ints.MinInt(wg.SX, sz.X)
	wg.SY = ints.MinInt(wg.SY, sz.Y)
	if wg.PX+wg.SX > sz.X {
		wg.PX = sz.X - wg.SX
	}
	if wg.PY+wg.SY > sz.Y {
		wg.PY = sz.Y - wg.SY
	}
}

// WindowGeomPrefs records the window geometry by window name, screen name --
// looks up the info automatically for new windows and saves persistently
type WindowGeomPrefs map[string]map[string]WindowGeom

// WinGeomPrefsFileName is the base name of the preferences file in GoGi prefs directory
var WinGeomPrefsFileName = "win_geom_prefs"

// WinGeomPrefsLastSave is when prefs were last saved -- if we weren't the last to save
// then we need to re-open before modifying
var WinGeomPrefsLastSave time.Time

// WinGeomPrefsMu is read-write mutex that protects updating of WinGeomPrefs
var WinGeomPrefsMu sync.RWMutex

var WinGeomPrefsLockSleep = 100 * time.Millisecond

var WinGeomNoLockErr = errors.New("WinGeom could not lock lock file")

// LockFile attempts to create the win_geom_prefs lock file
func (wg *WindowGeomPrefs) LockFile() error {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName+".lck")
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
			time.Sleep(WinGeomPrefsLockSleep)
			continue
		}
		var lts time.Time
		err = lts.UnmarshalJSON(b)
		if err != nil {
			time.Sleep(WinGeomPrefsLockSleep)
			continue
		}
		if time.Now().Sub(lts) > 1*time.Second {
			// log.Printf("WinGeomPrefs: lock file stale: %v\n", lts.String())
			os.Remove(pnm)
			continue
		}
		// log.Printf("WinGeomPrefs: waiting for lock file: %v\n", lts.String())
		time.Sleep(WinGeomPrefsLockSleep)
	}
	// log.Printf("WinGeomPrefs: failed to lock file: %v\n", pnm)
	return WinGeomNoLockErr
}

// UnLockFile unlocks the win_geom_prefs lock file (just removes it)
func (wg *WindowGeomPrefs) UnlockFile() {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName+".lck")
	os.Remove(pnm)
}

// NeedToReload returns true if the last save time of prefs file is more recent than
// when we last saved
func (wg *WindowGeomPrefs) NeedToReload() bool {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName+".lst")
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
	eq := lts.Equal(WinGeomPrefsLastSave)
	if !eq {
		// fmt.Printf("prefs file saved more recently: %v than our last save: %v\n", lts.String(),
		// 	WinGeomPrefsLastSave.String())
		WinGeomPrefsLastSave = lts
	}
	return !eq
}

// SaveLastSave saves timestamp (now) of last save to win geom
func (wg *WindowGeomPrefs) SaveLastSave() {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName+".lst")
	WinGeomPrefsLastSave = time.Now()
	b, _ := WinGeomPrefsLastSave.MarshalJSON()
	ioutil.WriteFile(pnm, b, 0644)
}

// Open Window Geom preferences from GoGi standard prefs directory
// called under mutex or at start
func (wg *WindowGeomPrefs) Open() error {
	if wg == nil {
		*wg = make(WindowGeomPrefs, 1000)
	}
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName+".json")
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		// log.Println(err)
		return err
	}
	err = json.Unmarshal(b, wg)
	if err != nil {
		log.Println(err)
	}
	oldFmt := false
	for _, wps := range *wg {
		for _, wp := range wps {
			if wp.DPI == 0 && wp.DPR == 0 {
				oldFmt = true
			}
			break
		}
		break
	}
	if oldFmt {
		log.Printf("WindowGeomPrefs: resetting prefs for new format\n")
		*wg = make(WindowGeomPrefs, 1000)
		wg.Save() // overwrite
	}
	return err
}

// Save Window Geom Preferences to GoGi standard prefs directory
// assumed to be under mutex and lock still
func (wg *WindowGeomPrefs) Save() error {
	if wg == nil {
		return nil
	}
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName+".json")
	b, err := json.MarshalIndent(wg, "", "\t")
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(pnm, b, 0644)
	if err != nil {
		log.Println(err)
	} else {
		wg.SaveLastSave()
	}
	return err
}

// RecordPref records current state of window as preference
func (wg *WindowGeomPrefs) RecordPref(win *Window) {
	if !win.IsVisible() {
		return
	}
	wsz := win.OSWin.WinSize()
	if wsz == image.ZP {
		if WinGeomTrace {
			fmt.Printf("WindowGeomPrefs: NOT storing null size for win: %v\n", win.Nm)
		}
		return
	}
	WinGeomPrefsMu.Lock()
	if wg == nil {
		*wg = make(WindowGeomPrefs, 100)
	}

	winName := win.Nm
	// only use the part of window name prior to colon -- that is the general "class" of window
	if ci := strings.Index(winName, ":"); ci > 0 {
		winName = winName[:ci]
	}
	sc := win.OSWin.Screen()
	wgr := WindowGeom{DPI: win.LogicalDPI(), DPR: sc.DevicePixelRatio}
	wgr.SetPos(win.OSWin.Position())
	wgr.SetSize(wsz)

	wg.LockFile() // not going to change our behavior if we can't lock!
	if wg.NeedToReload() {
		wg.Open()
	}

	if (*wg)[winName] == nil {
		(*wg)[winName] = make(map[string]WindowGeom, 10)
	}
	(*wg)[winName][sc.Name] = wgr
	if WinGeomTrace {
		fmt.Printf("WindowGeomPrefs Saving for window: %v pos: %v size: %v  screen: %v\n", winName, wgr.Pos(), wgr.Size(), sc.Name)
	}
	wg.Save()
	wg.UnlockFile()
	WinGeomPrefsMu.Unlock()
}

// Pref returns an existing preference for given window name, or one adapted
// to given screen if only records are on a different screen -- if scrn is nil
// then default (first) screen is used from oswin.TheApp
// if the window name has a colon, only the part prior to the colon is used
func (wg *WindowGeomPrefs) Pref(winName string, scrn *oswin.Screen) *WindowGeom {
	WinGeomPrefsMu.RLock()
	defer WinGeomPrefsMu.RUnlock()

	if wg == nil {
		return nil
	}
	// only use the part of window name prior to colon -- that is the general "class" of window
	if ci := strings.Index(winName, ":"); ci > 0 {
		winName = winName[:ci]
	}
	wps, ok := (*wg)[winName]
	if !ok {
		return nil
	}

	if scrn == nil {
		scrn = oswin.TheApp.Screen(0)
		if WinGeomTrace {
			fmt.Printf("WindowGeomPrefs: scrn is nil, using scrn 0: %v\n", scrn.Name)
		}
	}
	scsz := scrn.Geometry.Size()

	wp, ok := wps[scrn.Name]
	if ok {
		if scrn.LogicalDPI == wp.DPI && scrn.DevicePixelRatio == wp.DPR {
			wp.FitInSize(scsz)
			if WinGeomTrace {
				fmt.Printf("WindowGeomPrefs for window: %v size: %v  screen: %v\n", winName, wp.Size(), scrn.Name)
			}
			return &wp
		} else {
			if wp.DPI <= 0 {
				wp.DPI = 96
			}
			if WinGeomTrace {
				fmt.Printf("WindowGeomPrefs: rescaling scrn dpi: %v saved dpi: %v\n", scrn.LogicalDPI, wp.DPI)
			}
			fact := (wp.DPR / wp.DPI) / (scrn.DevicePixelRatio / scrn.LogicalDPI)
			wp.ScaleSize(fact)
			wp.ScalePos(fact)
			wp.FitInSize(scsz)
			if WinGeomTrace {
				fmt.Printf("WindowGeomPrefs for window: %v size: %v\n", winName, wp.Size())
			}
			return &wp
		}
	}

	if len(wps) == 0 { // shouldn't happen
		return nil
	}

	trgdpi := scrn.LogicalDPI / scrn.DevicePixelRatio
	if WinGeomTrace {
		fmt.Printf("WindowGeomPrefs: falling back on dpi conversion: %v\n", trgdpi)
	}

	// try to find one with same logical dpi, else closest
	var closest *WindowGeom
	minDPId := float32(100000.0)
	for _, wp = range wps {
		wdpi := wp.DPI / wp.DPR
		if wdpi == trgdpi {
			if WinGeomTrace {
				fmt.Printf("WindowGeomPrefs for window: %v other screen pos: %v size: %v\n", winName, wp.Pos(), wp.Size())
			}
			return &wp
		}
		dpid := mat32.Abs(wdpi - trgdpi)
		if dpid < minDPId {
			minDPId = dpid
			closest = &wp
		}
	}
	if closest == nil {
		return nil
	}
	wp = *closest

	fact := (closest.DPR / closest.DPI) / (scrn.DevicePixelRatio / scrn.LogicalDPI)
	wp.ScaleSize(fact)
	wp.ScalePos(fact)
	wp.FitInSize(scsz)
	if WinGeomTrace {
		fmt.Printf("WindowGeomPrefs for window: %v rescaled pos: %v size: %v\n   old dpi: %v cur dpi: %v  old size: %v\n", winName, wp.Pos(), wp.Size(), closest.DPI, scrn.LogicalDPI, closest.Size())
	}
	return &wp
}

// DeleteAll deletes the file that saves the position and size of each window,
// by screen, and clear current in-memory cache.  You shouldn't need to use
// this but sometimes useful for testing.
func (wg *WindowGeomPrefs) DeleteAll() {
	WinGeomPrefsMu.Lock()
	defer WinGeomPrefsMu.Unlock()

	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName+".json")
	os.Remove(pnm)
	*wg = make(WindowGeomPrefs, 1000)
}
