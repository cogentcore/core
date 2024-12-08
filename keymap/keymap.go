// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package keymap implements maps from keyboard shortcuts to
// semantic GUI keyboard functions.
package keymap

//go:generate core generate

import (
	"encoding/json"
	"log/slog"
	"slices"
	"sort"
	"strings"

	"cogentcore.org/core/events/key"
)

// https://en.wikipedia.org/wiki/Table_of_keyboard_shortcuts
// https://www.cs.colorado.edu/~main/cs1300/lab/emacs.html
// https://help.ubuntu.com/community/KeyboardShortcuts

// Functions are semantic functions that keyboard events
// can perform in the GUI.
type Functions int32 //enums:enum

const (
	None Functions = iota
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
	// WordLeft is the final navigation function -- all above also allow Shift+ for selection.

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
	MultiA    // multi-key sequence A: Emacs Control+C
	MultiB    // multi-key sequence B: Emacs Control+X
)

// Map is a map between a key sequence (chord) and a specific key
// function.  This mapping must be unique, in that each chord has a
// unique function, but multiple chords can trigger the same function.
type Map map[key.Chord]Functions

// ActiveMap points to the active map -- users can set this to an
// alternative map in Settings
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
	km, _, ok := AvailableMaps.MapByName(mapnm)
	if ok {
		SetActiveMap(km, mapnm)
	} else {
		slog.Error("keymap.SetActiveKeyMapName: key map named not found, using default", "requested", mapnm, "default", DefaultMap)
		km, _, ok = AvailableMaps.MapByName(DefaultMap)
		if ok {
			SetActiveMap(km, DefaultMap)
		} else {
			avail := make([]string, len(AvailableMaps))
			for i, km := range AvailableMaps {
				avail[i] = km.Name
			}
			slog.Error("keymap.SetActiveKeyMapName: DefaultKeyMap not found either; trying first one", "default", DefaultMap, "available", avail)
			if len(AvailableMaps) > 0 {
				nkm := AvailableMaps[0]
				SetActiveMap(&nkm.Map, MapName(nkm.Name))
			}
		}
	}
}

// Of converts the given [key.Chord] into a keyboard function.
func Of(chord key.Chord) Functions {
	f, ok := (*ActiveMap)[chord]
	if ok {
		return f
	}
	if strings.Contains(string(chord), "Shift+") {
		nsc := key.Chord(strings.ReplaceAll(string(chord), "Shift+", ""))
		if f, ok = (*ActiveMap)[nsc]; ok && f <= WordLeft { // automatically allow +Shift for nav
			return f
		}
	}
	return None
}

// MapItem records one element of the key map, which is used for organizing the map.
type MapItem struct {

	// the key chord that activates a function
	Key key.Chord

	// the function of that key
	Fun Functions
}

// ToSlice copies this keymap to a slice of [MapItem]s.
func (km *Map) ToSlice() []MapItem {
	kms := make([]MapItem, len(*km))
	idx := 0
	for key, fun := range *km {
		kms[idx] = MapItem{key, fun}
		idx++
	}
	return kms
}

// ChordFor returns all of the key chord triggers for the given
// key function in the map, separating them with newlines.
func (km *Map) ChordFor(kf Functions) key.Chord {
	res := []string{}
	for key, fun := range *km {
		if fun == kf {
			res = append(res, string(key))
		}
	}
	slices.Sort(res)
	return key.Chord(strings.Join(res, "\n"))
}

// Chord returns all of the key chord triggers for this
// key function in the current active map, separating them with newlines.
func (kf Functions) Chord() key.Chord {
	return ActiveMap.ChordFor(kf)
}

// Label transforms the key function into a string representing
// its underlying key chord(s) in a form suitable for display to users.
func (kf Functions) Label() string {
	return kf.Chord().Label()
}

// Update ensures that the given keymap has at least one entry for every
// defined key function, grabbing ones from the default map if not, and
// also eliminates any [None] entries which might reflect out-of-date
// functions.
func (km *Map) Update(kmName MapName) {
	for key, val := range *km {
		if val == None {
			slog.Error("keymap.KeyMap: key function is nil; probably renamed", "key", key)
			delete(*km, key)
		}
	}
	kms := km.ToSlice()
	addkm := make([]MapItem, 0)

	sort.Slice(kms, func(i, j int) bool {
		return kms[i].Fun < kms[j].Fun
	})

	lfun := None
	for _, ki := range kms {
		fun := ki.Fun
		if fun != lfun {
			del := fun - lfun
			if del > 1 {
				for mi := lfun + 1; mi < fun; mi++ {
					// slog.Error("keymap.KeyMap: key map is missing a key for a key function", "keyMap", kmName, "function", mi)
					s := mi.String()
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

// DefaultMap is the overall default keymap, which is set in init
// depending on the platform
var DefaultMap MapName = "LinuxStandard"

// MapsItem is an entry in a Maps list
type MapsItem struct { //types:add -setters

	// name of keymap
	Name string `width:"20"`

	// description of keymap; good idea to include source it was derived from
	Desc string

	// to edit key sequence click button and type new key combination; to edit function mapped to key sequence choose from menu
	Map Map
}

// Label satisfies the Labeler interface
func (km MapsItem) Label() string {
	return km.Name
}

// Maps is a list of [MapsItem]s; users can edit these in their settings.
type Maps []MapsItem //types:add

// AvailableMaps is the current list of available keymaps for use.
// This can be loaded / saved / edited in user settings. This is set
// to [StandardMaps] at startup.
var AvailableMaps Maps

// MapByName returns a [Map] and index by name. It returns false
// and prints an error message if not found.
func (km *Maps) MapByName(name MapName) (*Map, int, bool) {
	for i, it := range *km {
		if it.Name == string(name) {
			return &it.Map, i, true
		}
	}
	slog.Error("keymap.KeyMaps.MapByName: key map not found", "name", name)
	return nil, -1, false
}

// CopyFrom copies keymaps from given other map
func (km *Maps) CopyFrom(cp Maps) {
	*km = make(Maps, 0, len(cp)) // reset
	b, _ := json.Marshal(cp)
	json.Unmarshal(b, km)
}

// MergeFrom merges keymaps from given other map
func (km *Maps) MergeFrom(cp Maps) {
	for nm, mi := range cp {
		tmi := (*km)[nm]
		for ch, kf := range mi.Map {
			tmi.Map[ch] = kf
		}
	}
}

// order is: Shift, Control, Alt, Meta
// note: shift and meta modifiers for navigation keys do select + move

// note: where multiple shortcuts exist for a given function, any shortcut
// display of such items in menus will randomly display one of the
// options. This can be considered a feature, not a bug!
