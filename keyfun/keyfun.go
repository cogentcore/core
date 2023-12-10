// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keyfun

//go:generate goki generate

import (
	"encoding/json"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"goki.dev/goosi"
	"goki.dev/goosi/events/key"
	"goki.dev/grows/jsons"
	"goki.dev/grr"
)

// https://en.wikipedia.org/wiki/Table_of_keyboard_shortcuts
// https://www.cs.colorado.edu/~main/cs1300/lab/emacs.html
// https://help.ubuntu.com/community/KeyboardShortcuts

// Funs are functions that keyboard events can perform in the GUI.
// It seems possible to keep this flat and consistent across different contexts,
// as long as the functions can be appropriately reinterpreted for each context.
type Funs int32 //enums:enum

const (
	Nil Funs = iota
	MoveUp
	MoveDown
	MoveRight
	MoveLeft
	PageUp
	PageDown
	// PageRight
	// PageLeft
	Home    // start-of-line
	End     // end-of-line
	DocHome // start-of-doc -- Control / Alt / Shift +Home
	DocEnd  // end-of-doc Control / Alt / Shift +End
	WordRight
	WordLeft
	FocusNext // Tab
	FocusPrev // Shift-Tab
	Enter     // Enter / return key -- has various special functions
	Accept    // Ctrl+Enter = accept any changes and close dialog / move to next
	CancelSelect
	SelectMode
	SelectAll
	Abort
	// EditItem
	Copy
	Cut
	Paste
	PasteHist // from history
	Backspace
	BackspaceWord
	Delete
	DeleteWord
	Kill
	Duplicate
	Transpose
	TransposeWord
	Undo
	Redo
	Insert
	InsertAfter
	ZoomOut
	ZoomIn
	Prefs
	Refresh
	Recenter // Ctrl+L in emacs
	Complete
	Lookup
	Search // Ctrl+S in emacs -- more interactive type of search
	Find   // Command+F full-dialog find
	Replace
	Jump // jump to line
	HistPrev
	HistNext
	Menu // put focus on menu
	WinFocusNext
	WinClose
	WinSnapshot
	Inspector
	// Below are menu specific functions -- use these as shortcuts for menu buttons
	// allows uniqueness of mapping and easy customization of all key buttons
	New
	NewAlt1 // alternative version (e.g., shift)
	NewAlt2 // alternative version (e.g., alt)
	Open
	OpenAlt1 // alternative version (e.g., shift)
	OpenAlt2 // alternative version (e.g., alt)
	Save
	SaveAs
	SaveAlt   // another alt (e.g., alt)
	CloseAlt1 // alternative version (e.g., shift)
	CloseAlt2 // alternative version (e.g., alt)
)

// Map is a map between a key sequence (chord) and a specific KeyFun
// function.  This mapping must be unique, in that each chord has unique
// KeyFun, but multiple chords can trigger the same function.
type Map map[key.Chord]Funs

// ActiveMap points to the active map -- users can set this to an
// alternative map in Prefs
var ActiveMap *Map

// MapName has an associated Value for selecting from the list of
// available key map names, for use in preferences etc.
type MapName string

// ActiveMapName is the name of the active keymap
var ActiveMapName MapName

// SetActiveMap sets the current ActiveKeyMap, calling Update on the map
// prior to setting it to ensure that it is a valid, complete map
func SetActiveMap(km *Map, kmName MapName) {
	km.Update(kmName)
	ActiveMap = km
	ActiveMapName = kmName
}

// SetActiveMapName sets the current ActiveKeyMap by name from those
// defined in AvailKeyMaps, calling Update on the map prior to setting it to
// ensure that it is a valid, complete map
func SetActiveMapName(mapnm MapName) {
	km, _, ok := AvailMaps.MapByName(mapnm)
	if ok {
		SetActiveMap(km, mapnm)
	} else {
		slog.Error("keyfun.SetActiveKeyMapName: key map named not found, using default", "requested", mapnm, "default", DefaultMap)
		km, _, ok = AvailMaps.MapByName(DefaultMap)
		if ok {
			SetActiveMap(km, DefaultMap)
		} else {
			avail := make([]string, len(AvailMaps))
			for i, km := range AvailMaps {
				avail[i] = km.Name
			}
			slog.Error("keyfun.SetActiveKeyMapName: DefaultKeyMap not found either; trying first one", "default", DefaultMap, "available", avail)
			if len(AvailMaps) > 0 {
				nkm := AvailMaps[0]
				SetActiveMap(&nkm.Map, MapName(nkm.Name))
			}
		}
	}
}

// Of translates chord into keyboard function -- use goosi key.Chord
// to get chord
func Of(chord key.Chord) Funs {
	kf := Nil
	if chord != "" {
		kf = (*ActiveMap)[chord]
		// if KeyEventTrace {
		// 	fmt.Printf("keyfun.KeyFun chord: %v = %v\n", chord, kf)
		// }
	}
	return kf
}

// MapItem records one element of the key map -- used for organizing the map.
type MapItem struct {

	// the key chord that activates a function
	Key key.Chord

	// the function of that key
	Fun Funs
}

// ToSlice copies this keymap to a slice of KeyMapItem's
func (km *Map) ToSlice() []MapItem {
	kms := make([]MapItem, len(*km))
	idx := 0
	for key, fun := range *km {
		kms[idx] = MapItem{key, fun}
		idx++
	}
	return kms
}

// ChordForFun returns first key chord trigger for given KeyFun in map
func (km *Map) ChordFor(kf Funs) key.Chord {
	for key, fun := range *km {
		if fun == kf {
			return key
		}
	}
	return ""
}

// ChordForFun returns first key chord trigger for given KeyFun in the
// current active map
func ChordFor(kf Funs) key.Chord {
	return ActiveMap.ChordFor(kf)
}

// ShortcutForFun returns OS-specific formatted shortcut for first key chord
// trigger for given KeyFun in map
func (km *Map) ShortcutFor(kf Funs) key.Chord {
	return km.ChordFor(kf).OSShortcut()
}

// ShortcutFor returns OS-specific formatted shortcut for first key chord
// trigger for given KeyFun in the current active map
func ShortcutFor(kf Funs) key.Chord {
	return ActiveMap.ShortcutFor(kf)
}

// Update ensures that the given keymap has at least one entry for every
// defined KeyFun, grabbing ones from the default map if not, and also
// eliminates any Nil entries which might reflect out-of-date functions
func (km *Map) Update(kmName MapName) {
	for key, val := range *km {
		if val == Nil {
			slog.Error("keyfun.KeyMap: key function is nil; probably renamed", "key", key)
			delete(*km, key)
		}
	}
	kms := km.ToSlice()
	addkm := make([]MapItem, 0)

	sort.Slice(kms, func(i, j int) bool {
		return kms[i].Fun < kms[j].Fun
	})

	lfun := Nil
	for _, ki := range kms {
		fun := ki.Fun
		if fun != lfun {
			del := fun - lfun
			if del > 1 {
				for mi := lfun + 1; mi < fun; mi++ {
					slog.Error("keyfun.KeyMap: key map is missing a key for a key function", "keyMap", kmName, "function", mi)
					s := mi.String()
					s = strings.TrimPrefix(s, "KeyFun")
					s = "- Not Set - " + s
					nski := MapItem{Key: key.Chord(s), Fun: mi}
					addkm = append(addkm, nski)
				}
			}
			lfun = fun
		}
	}

	for _, ai := range addkm {
		(*km)[ai.Key] = ai.Fun
	}
}

/////////////////////////////////////////////////////////////////////////////////
// KeyMaps -- list of KeyMap's

// DefaultMap is the overall default keymap -- reinitialized in gimain init()
// depending on platform
var DefaultMap = MapName("MacEmacs")

// MapsItem is an entry in a Maps list
type MapsItem struct { //gti:add -setters

	// name of keymap
	Name string `width:"20"`

	// description of keymap -- good idea to include source it was derived from
	Desc string

	// to edit key sequence click button and type new key combination; to edit function mapped to key sequence choose from menu
	Map Map
}

// Label satisfies the Labeler interface
func (km MapsItem) Label() string {
	return km.Name
}

// Maps is a list of KeyMap's -- users can edit these in Prefs -- to create
// a custom one, just duplicate an existing map, rename, and customize
type Maps []MapsItem //gti:add

// AvailMaps is the current list of available keymaps for use -- can be
// loaded / saved / edited with preferences.  This is set to StdKeyMaps at
// startup.
var AvailMaps Maps

func init() {
	AvailMaps.CopyFrom(StdMaps)
}

// MapByName returns a keymap and index by name -- returns false and emits a
// message to stdout if not found
func (km *Maps) MapByName(name MapName) (*Map, int, bool) {
	for i, it := range *km {
		if it.Name == string(name) {
			return &it.Map, i, true
		}
	}
	slog.Error("keyfun.KeyMaps.MapByName: key map not found", "name", name)
	return nil, -1, false
}

// PrefsMapsFileName is the name of the preferences file in GoGi prefs
// directory for saving / loading the default AvailMaps key maps list
var PrefsMapsFileName = "key_maps_prefs.json"

// Open opens keymaps from a json-formatted file.
// You can save and open key maps to / from files to share, experiment, transfer, etc
func (km *Maps) Open(filename string) error { //gti:add
	*km = make(Maps, 0, 10) // reset
	return grr.Log(jsons.Open(km, filename))
}

// Save saves keymaps to a json-formatted file.
// You can save and open key maps to / from files to share, experiment, transfer, etc
func (km *Maps) Save(filename string) error { //gti:add
	return grr.Log(jsons.Save(km, filename))
}

// OpenPrefs opens KeyMaps from GoGi standard prefs directory, in file key_maps_prefs.json.
// This is called automatically, so calling it manually should not be necessary in most cases.
func (km *Maps) OpenPrefs() error { //gti:add
	pdir := goosi.TheApp.GoGiDataDir()
	pnm := filepath.Join(pdir, PrefsMapsFileName)
	AvailMapsChanged = false
	return km.Open(pnm)
}

// SavePrefs saves KeyMaps to GoGi standard prefs directory, in file key_maps_prefs.json,
// which will be loaded automatically at startup if prefs SaveKeyMaps is checked
// (should be if you're using custom keymaps)
func (km *Maps) SavePrefs() error { //gti:add
	pdir := goosi.TheApp.GoGiDataDir()
	pnm := filepath.Join(pdir, PrefsMapsFileName)
	AvailMapsChanged = false
	return km.Save(pnm)
}

// CopyFrom copies keymaps from given other map
func (km *Maps) CopyFrom(cp Maps) {
	*km = make(Maps, 0, len(cp)) // reset
	b, _ := json.Marshal(cp)
	json.Unmarshal(b, km)
}

// RevertToStd reverts the keymaps to using the StdKeyMaps that are compiled into the program
// and have all the lastest key functions defined.  If you have edited your maps, and are finding
// things not working, it is a good idea to save your current maps and try this, or at least do
// ViewStdMaps to see the current standards. Your current map edits will be lost if you proceed!
func (km *Maps) RevertToStd() { //gti:add
	km.CopyFrom(StdMaps)
	AvailMapsChanged = true
}

// AvailMapsChanged is used to update giv.KeyMapsView toolbars via
// following menu, toolbar props update methods -- not accurate if editing any
// other map but works for now..
var AvailMapsChanged = false

// order is: Shift, Control, Alt, Meta
// note: shift and meta modifiers for navigation keys do select + move

// note: where multiple shortcuts exist for a given function, any shortcut
// display of such items in menus will randomly display one of the
// options. This can be considered a feature, not a bug!
