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

func (kn MapName) String() string {
	return string(kn)
}

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

// Of translates chord into keyboard function -- use oswin key.Chord
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
	pdir := goosi.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, PrefsMapsFileName)
	AvailMapsChanged = false
	return km.Open(pnm)
}

// SavePrefs saves KeyMaps to GoGi standard prefs directory, in file key_maps_prefs.json,
// which will be loaded automatically at startup if prefs SaveKeyMaps is checked
// (should be if you're using custom keymaps)
func (km *Maps) SavePrefs() error { //gti:add
	pdir := goosi.TheApp.GoGiPrefsDir()
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

// StdMaps is the original compiled-in set of standard keymaps that have
// the lastest key functions bound to standard key chords.
var StdMaps = Maps{
	{"MacStd", "Standard Mac KeyMap", Map{
		"UpArrow":                 MoveUp,
		"Shift+UpArrow":           MoveUp,
		"Meta+UpArrow":            MoveUp,
		"Control+P":               MoveUp,
		"Shift+Control+P":         MoveUp,
		"Meta+Control+P":          MoveUp,
		"DownArrow":               MoveDown,
		"Shift+DownArrow":         MoveDown,
		"Meta+DownArrow":          MoveDown,
		"Control+N":               MoveDown,
		"Shift+Control+N":         MoveDown,
		"Meta+Control+N":          MoveDown,
		"RightArrow":              MoveRight,
		"Shift+RightArrow":        MoveRight,
		"Meta+RightArrow":         End,
		"Control+F":               MoveRight,
		"Shift+Control+F":         MoveRight,
		"Meta+Control+F":          MoveRight,
		"LeftArrow":               MoveLeft,
		"Shift+LeftArrow":         MoveLeft,
		"Meta+LeftArrow":          Home,
		"Control+B":               MoveLeft,
		"Shift+Control+B":         MoveLeft,
		"Meta+Control+B":          MoveLeft,
		"PageUp":                  PageUp,
		"Shift+PageUp":            PageUp,
		"Control+UpArrow":         PageUp,
		"Control+U":               PageUp,
		"PageDown":                PageDown,
		"Shift+PageDown":          PageDown,
		"Control+DownArrow":       PageDown,
		"Shift+Control+V":         PageDown,
		"Alt+√":                   PageDown,
		"Meta+Home":               DocHome,
		"Shift+Home":              DocHome,
		"Meta+H":                  DocHome,
		"Meta+End":                DocEnd,
		"Shift+End":               DocEnd,
		"Meta+L":                  DocEnd,
		"Control+RightArrow":      WordRight,
		"Control+LeftArrow":       WordLeft,
		"Alt+RightArrow":          WordRight,
		"Shift+Alt+RightArrow":    WordRight,
		"Alt+LeftArrow":           WordLeft,
		"Shift+Alt+LeftArrow":     WordLeft,
		"Home":                    Home,
		"Control+A":               Home,
		"Shift+Control+A":         Home,
		"End":                     End,
		"Control+E":               End,
		"Shift+Control+E":         End,
		"Tab":                     FocusNext,
		"Shift+Tab":               FocusPrev,
		"ReturnEnter":             Enter,
		"KeypadEnter":             Enter,
		"Meta+A":                  SelectAll,
		"Control+G":               CancelSelect,
		"Control+Spacebar":        SelectMode,
		"Control+ReturnEnter":     Accept,
		"Escape":                  Abort,
		"DeleteBackspace":         Backspace,
		"Control+DeleteBackspace": BackspaceWord,
		"Alt+DeleteBackspace":     BackspaceWord,
		"DeleteForward":           Delete,
		"Control+DeleteForward":   DeleteWord,
		"Alt+DeleteForward":       DeleteWord,
		"Control+D":               Delete,
		"Control+K":               Kill,
		"Alt+∑":                   Copy,
		"Meta+C":                  Copy,
		"Control+W":               Cut,
		"Meta+X":                  Cut,
		"Control+Y":               Paste,
		"Control+V":               Paste,
		"Meta+V":                  Paste,
		"Shift+Meta+V":            PasteHist,
		"Alt+D":                   Duplicate,
		"Control+T":               Transpose,
		"Alt+T":                   TransposeWord,
		"Control+Z":               Undo,
		"Meta+Z":                  Undo,
		"Shift+Control+Z":         Redo,
		"Shift+Meta+Z":            Redo,
		"Control+I":               Insert,
		"Control+O":               InsertAfter,
		"Shift+Meta+=":            ZoomIn,
		"Meta+=":                  ZoomIn,
		"Meta+-":                  ZoomOut,
		"Control+=":               ZoomIn,
		"Shift+Control++":         ZoomIn,
		"Shift+Meta+-":            ZoomOut,
		"Control+-":               ZoomOut,
		"Shift+Control+_":         ZoomOut,
		"Control+Alt+P":           Prefs,
		"F5":                      Refresh,
		"Control+L":               Recenter,
		"Control+.":               Complete,
		"Control+,":               Lookup,
		"Control+S":               Search,
		"Meta+F":                  Find,
		"Meta+R":                  Replace,
		"Control+J":               Jump,
		"Control+[":               HistPrev,
		"Control+]":               HistNext,
		"Meta+[":                  HistPrev,
		"Meta+]":                  HistNext,
		"F10":                     Menu,
		"Meta+`":                  WinFocusNext,
		"Meta+W":                  WinClose,
		"Control+Alt+G":           WinSnapshot,
		"Shift+Control+G":         WinSnapshot,
		"Control+Alt+I":           Inspector,
		"Shift+Control+I":         Inspector,
		"Meta+N":                  New,
		"Shift+Meta+N":            NewAlt1,
		"Alt+Meta+N":              NewAlt2,
		"Meta+O":                  Open,
		"Shift+Meta+O":            OpenAlt1,
		"Alt+Meta+O":              OpenAlt2,
		"Meta+S":                  Save,
		"Shift+Meta+S":            SaveAs,
		"Alt+Meta+S":              SaveAlt,
		"Shift+Meta+W":            CloseAlt1,
		"Alt+Meta+W":              CloseAlt2,
	}},
	{"MacEmacs", "Mac with emacs-style navigation -- emacs wins in conflicts", Map{
		"UpArrow":                 MoveUp,
		"Shift+UpArrow":           MoveUp,
		"Meta+UpArrow":            MoveUp,
		"Control+P":               MoveUp,
		"Shift+Control+P":         MoveUp,
		"Meta+Control+P":          MoveUp,
		"DownArrow":               MoveDown,
		"Shift+DownArrow":         MoveDown,
		"Meta+DownArrow":          MoveDown,
		"Control+N":               MoveDown,
		"Shift+Control+N":         MoveDown,
		"Meta+Control+N":          MoveDown,
		"RightArrow":              MoveRight,
		"Shift+RightArrow":        MoveRight,
		"Meta+RightArrow":         End,
		"Control+F":               MoveRight,
		"Shift+Control+F":         MoveRight,
		"Meta+Control+F":          MoveRight,
		"LeftArrow":               MoveLeft,
		"Shift+LeftArrow":         MoveLeft,
		"Meta+LeftArrow":          Home,
		"Control+B":               MoveLeft,
		"Shift+Control+B":         MoveLeft,
		"Meta+Control+B":          MoveLeft,
		"PageUp":                  PageUp,
		"Shift+PageUp":            PageUp,
		"Control+UpArrow":         PageUp,
		"Control+U":               PageUp,
		"PageDown":                PageDown,
		"Shift+PageDown":          PageDown,
		"Control+DownArrow":       PageDown,
		"Shift+Control+V":         PageDown,
		"Alt+√":                   PageDown,
		"Control+V":               PageDown,
		"Control+RightArrow":      WordRight,
		"Control+LeftArrow":       WordLeft,
		"Alt+RightArrow":          WordRight,
		"Shift+Alt+RightArrow":    WordRight,
		"Alt+LeftArrow":           WordLeft,
		"Shift+Alt+LeftArrow":     WordLeft,
		"Home":                    Home,
		"Control+A":               Home,
		"Shift+Control+A":         Home,
		"End":                     End,
		"Control+E":               End,
		"Shift+Control+E":         End,
		"Meta+Home":               DocHome,
		"Shift+Home":              DocHome,
		"Meta+H":                  DocHome,
		"Control+H":               DocHome,
		"Control+Alt+A":           DocHome,
		"Meta+End":                DocEnd,
		"Shift+End":               DocEnd,
		"Meta+L":                  DocEnd,
		"Control+Alt+E":           DocEnd,
		"Alt+Ƒ":                   WordRight,
		"Alt+∫":                   WordLeft,
		"Tab":                     FocusNext,
		"Shift+Tab":               FocusPrev,
		"ReturnEnter":             Enter,
		"KeypadEnter":             Enter,
		"Meta+A":                  SelectAll,
		"Control+G":               CancelSelect,
		"Control+Spacebar":        SelectMode,
		"Control+ReturnEnter":     Accept,
		"Escape":                  Abort,
		"DeleteBackspace":         Backspace,
		"Control+DeleteBackspace": BackspaceWord,
		"Alt+DeleteBackspace":     BackspaceWord,
		"DeleteForward":           Delete,
		"Control+DeleteForward":   DeleteWord,
		"Alt+DeleteForward":       DeleteWord,
		"Control+D":               Delete,
		"Control+K":               Kill,
		"Alt+∑":                   Copy,
		"Meta+C":                  Copy,
		"Control+W":               Cut,
		"Meta+X":                  Cut,
		"Control+Y":               Paste,
		"Meta+V":                  Paste,
		"Shift+Meta+V":            PasteHist,
		"Shift+Control+Y":         PasteHist,
		"Alt+∂":                   Duplicate,
		"Control+T":               Transpose,
		"Alt+T":                   TransposeWord,
		"Control+Z":               Undo,
		"Meta+Z":                  Undo,
		"Control+/":               Undo,
		"Shift+Control+Z":         Redo,
		"Shift+Meta+Z":            Redo,
		"Control+I":               Insert,
		"Control+O":               InsertAfter,
		"Shift+Meta+=":            ZoomIn,
		"Meta+=":                  ZoomIn,
		"Meta+-":                  ZoomOut,
		"Control+=":               ZoomIn,
		"Shift+Control++":         ZoomIn,
		"Shift+Meta+-":            ZoomOut,
		"Control+-":               ZoomOut,
		"Shift+Control+_":         ZoomOut,
		"Control+Alt+P":           Prefs,
		"F5":                      Refresh,
		"Control+L":               Recenter,
		"Control+.":               Complete,
		"Control+,":               Lookup,
		"Control+S":               Search,
		"Meta+F":                  Find,
		"Meta+R":                  Replace,
		"Control+R":               Replace,
		"Control+J":               Jump,
		"Control+[":               HistPrev,
		"Control+]":               HistNext,
		"Meta+[":                  HistPrev,
		"Meta+]":                  HistNext,
		"F10":                     Menu,
		"Meta+`":                  WinFocusNext,
		"Meta+W":                  WinClose,
		"Control+Alt+G":           WinSnapshot,
		"Shift+Control+G":         WinSnapshot,
		"Control+Alt+I":           Inspector,
		"Shift+Control+I":         Inspector,
		"Meta+N":                  New,
		"Shift+Meta+N":            NewAlt1,
		"Alt+Meta+N":              NewAlt2,
		"Meta+O":                  Open,
		"Shift+Meta+O":            OpenAlt1,
		"Alt+Meta+O":              OpenAlt2,
		"Meta+S":                  Save,
		"Shift+Meta+S":            SaveAs,
		"Alt+Meta+S":              SaveAlt,
		"Shift+Meta+W":            CloseAlt1,
		"Alt+Meta+W":              CloseAlt2,
	}},
	{"LinuxEmacs", "Linux with emacs-style navigation -- emacs wins in conflicts", Map{
		"UpArrow":                 MoveUp,
		"Shift+UpArrow":           MoveUp,
		"Alt+UpArrow":             MoveUp,
		"Control+P":               MoveUp,
		"Shift+Control+P":         MoveUp,
		"Alt+Control+P":           MoveUp,
		"DownArrow":               MoveDown,
		"Shift+DownArrow":         MoveDown,
		"Alt+DownArrow":           MoveDown,
		"Control+N":               MoveDown,
		"Shift+Control+N":         MoveDown,
		"Alt+Control+N":           MoveDown,
		"RightArrow":              MoveRight,
		"Shift+RightArrow":        MoveRight,
		"Alt+RightArrow":          End,
		"Control+F":               MoveRight,
		"Shift+Control+F":         MoveRight,
		"Alt+Control+F":           MoveRight,
		"LeftArrow":               MoveLeft,
		"Shift+LeftArrow":         MoveLeft,
		"Alt+LeftArrow":           Home,
		"Control+B":               MoveLeft,
		"Shift+Control+B":         MoveLeft,
		"Alt+Control+B":           MoveLeft,
		"PageUp":                  PageUp,
		"Shift+PageUp":            PageUp,
		"Control+UpArrow":         PageUp,
		"Control+U":               PageUp,
		"Shift+Control+U":         PageUp,
		"Alt+Control+U":           PageUp,
		"PageDown":                PageDown,
		"Shift+PageDown":          PageDown,
		"Control+DownArrow":       PageDown,
		"Control+V":               PageDown,
		"Shift+Control+V":         PageDown,
		"Alt+Control+V":           PageDown,
		"Alt+Home":                DocHome,
		"Shift+Home":              DocHome,
		"Alt+H":                   DocHome,
		"Control+Alt+A":           DocHome,
		"Alt+End":                 DocEnd,
		"Shift+End":               DocEnd,
		"Alt+L":                   DocEnd,
		"Control+Alt+E":           DocEnd,
		"Control+RightArrow":      WordRight,
		"Control+LeftArrow":       WordLeft,
		"Home":                    Home,
		"Control+A":               Home,
		"Shift+Control+A":         Home,
		"End":                     End,
		"Control+E":               End,
		"Shift+Control+E":         End,
		"Tab":                     FocusNext,
		"Shift+Tab":               FocusPrev,
		"ReturnEnter":             Enter,
		"KeypadEnter":             Enter,
		"Alt+A":                   SelectAll,
		"Control+G":               CancelSelect,
		"Control+Spacebar":        SelectMode,
		"Control+ReturnEnter":     Accept,
		"Escape":                  Abort,
		"DeleteBackspace":         Backspace,
		"Control+DeleteBackspace": BackspaceWord,
		"DeleteForward":           Delete,
		"Control+D":               Delete,
		"Control+DeleteForward":   DeleteWord,
		"Alt+DeleteForward":       DeleteWord,
		"Control+K":               Kill,
		"Alt+W":                   Copy,
		"Alt+C":                   Copy,
		"Control+W":               Cut,
		"Alt+X":                   Cut,
		"Control+Y":               Paste,
		"Alt+V":                   Paste,
		"Shift+Alt+V":             PasteHist,
		"Shift+Control+Y":         PasteHist,
		"Alt+D":                   Duplicate,
		"Control+T":               Transpose,
		"Alt+T":                   TransposeWord,
		"Control+Z":               Undo,
		"Control+/":               Undo,
		"Shift+Control+Z":         Redo,
		"Control+I":               Insert,
		"Control+O":               InsertAfter,
		"Control+=":               ZoomIn,
		"Shift+Control++":         ZoomIn,
		"Control+-":               ZoomOut,
		"Shift+Control+_":         ZoomOut,
		"Control+Alt+P":           Prefs,
		"F5":                      Refresh,
		"Control+L":               Recenter,
		"Control+.":               Complete,
		"Control+,":               Lookup,
		"Control+S":               Search,
		"Alt+F":                   Find,
		"Control+R":               Replace,
		"Control+J":               Jump,
		"Control+[":               HistPrev,
		"Control+]":               HistNext,
		"F10":                     Menu,
		"Alt+F6":                  WinFocusNext,
		"Shift+Control+W":         WinClose,
		"Control+Alt+G":           WinSnapshot,
		"Shift+Control+G":         WinSnapshot,
		"Control+Alt+I":           Inspector,
		"Shift+Control+I":         Inspector,
		"Alt+N":                   New, // ctrl keys conflict..
		"Shift+Alt+N":             NewAlt1,
		"Control+Alt+N":           NewAlt2,
		"Alt+O":                   Open,
		"Shift+Alt+O":             OpenAlt1,
		"Control+Alt+O":           OpenAlt2,
		"Alt+S":                   Save,
		"Shift+Alt+S":             SaveAs,
		"Control+Alt+S":           SaveAlt,
		"Shift+Alt+W":             CloseAlt1,
		"Control+Alt+W":           CloseAlt2,
	}},
	{"LinuxStd", "Standard Linux KeyMap", Map{
		"UpArrow":                 MoveUp,
		"Shift+UpArrow":           MoveUp,
		"DownArrow":               MoveDown,
		"Shift+DownArrow":         MoveDown,
		"RightArrow":              MoveRight,
		"Shift+RightArrow":        MoveRight,
		"LeftArrow":               MoveLeft,
		"Shift+LeftArrow":         MoveLeft,
		"PageUp":                  PageUp,
		"Shift+PageUp":            PageUp,
		"Control+UpArrow":         PageUp,
		"PageDown":                PageDown,
		"Shift+PageDown":          PageDown,
		"Control+DownArrow":       PageDown,
		"Home":                    Home,
		"Alt+LeftArrow":           Home,
		"End":                     End,
		"Alt+Home":                DocHome,
		"Shift+Home":              DocHome,
		"Alt+End":                 DocEnd,
		"Shift+End":               DocEnd,
		"Control+RightArrow":      WordRight,
		"Control+LeftArrow":       WordLeft,
		"Alt+RightArrow":          End,
		"Tab":                     FocusNext,
		"Shift+Tab":               FocusPrev,
		"ReturnEnter":             Enter,
		"KeypadEnter":             Enter,
		"Control+A":               SelectAll,
		"Shift+Control+A":         CancelSelect,
		"Control+G":               CancelSelect,
		"Control+Spacebar":        SelectMode, // change input method / keyboard
		"Control+ReturnEnter":     Accept,
		"Escape":                  Abort,
		"DeleteBackspace":         Backspace,
		"Control+DeleteBackspace": BackspaceWord,
		"DeleteForward":           Delete,
		"Control+DeleteForward":   DeleteWord,
		"Alt+DeleteForward":       DeleteWord,
		"Control+K":               Kill,
		"Control+C":               Copy,
		"Control+X":               Cut,
		"Control+V":               Paste,
		"Shift+Control+V":         PasteHist,
		"Alt+D":                   Duplicate,
		"Control+T":               Transpose,
		"Alt+T":                   TransposeWord,
		"Control+Z":               Undo,
		"Control+Y":               Redo,
		"Shift+Control+Z":         Redo,
		"Control+Alt+I":           Insert,
		"Control+Alt+O":           InsertAfter,
		"Control+=":               ZoomIn,
		"Shift+Control++":         ZoomIn,
		"Control+-":               ZoomOut,
		"Shift+Control+_":         ZoomOut,
		"Shift+Control+P":         Prefs,
		"Control+Alt+P":           Prefs,
		"F5":                      Refresh,
		"Control+L":               Recenter,
		"Control+.":               Complete,
		"Control+,":               Lookup,
		"Alt+S":                   Search,
		"Control+F":               Find,
		"Control+H":               Replace,
		"Control+R":               Replace,
		"Control+J":               Jump,
		"Control+[":               HistPrev,
		"Control+]":               HistNext,
		"Control+N":               New,
		"F10":                     Menu,
		"Alt+F6":                  WinFocusNext,
		"Control+W":               WinClose,
		"Control+Alt+G":           WinSnapshot,
		"Shift+Control+G":         WinSnapshot,
		"Shift+Control+I":         Inspector,
		"Shift+Control+N":         NewAlt1,
		"Control+Alt+N":           NewAlt2,
		"Control+O":               Open,
		"Shift+Control+O":         OpenAlt1,
		"Shift+Alt+O":             OpenAlt2,
		"Control+S":               Save,
		"Shift+Control+S":         SaveAs,
		"Control+Alt+S":           SaveAlt,
		"Shift+Control+W":         CloseAlt1,
		"Control+Alt+W":           CloseAlt2,
	}},
	{"WindowsStd", "Standard Windows KeyMap", Map{
		"UpArrow":                 MoveUp,
		"Shift+UpArrow":           MoveUp,
		"DownArrow":               MoveDown,
		"Shift+DownArrow":         MoveDown,
		"RightArrow":              MoveRight,
		"Shift+RightArrow":        MoveRight,
		"LeftArrow":               MoveLeft,
		"Shift+LeftArrow":         MoveLeft,
		"PageUp":                  PageUp,
		"Shift+PageUp":            PageUp,
		"Control+UpArrow":         PageUp,
		"PageDown":                PageDown,
		"Shift+PageDown":          PageDown,
		"Control+DownArrow":       PageDown,
		"Home":                    Home,
		"Alt+LeftArrow":           Home,
		"End":                     End,
		"Alt+RightArrow":          End,
		"Alt+Home":                DocHome,
		"Shift+Home":              DocHome,
		"Alt+End":                 DocEnd,
		"Shift+End":               DocEnd,
		"Control+RightArrow":      WordRight,
		"Control+LeftArrow":       WordLeft,
		"Tab":                     FocusNext,
		"Shift+Tab":               FocusPrev,
		"ReturnEnter":             Enter,
		"KeypadEnter":             Enter,
		"Control+A":               SelectAll,
		"Shift+Control+A":         CancelSelect,
		"Control+G":               CancelSelect,
		"Control+Spacebar":        SelectMode, // change input method / keyboard
		"Control+ReturnEnter":     Accept,
		"Escape":                  Abort,
		"DeleteBackspace":         Backspace,
		"Control+DeleteBackspace": BackspaceWord,
		"DeleteForward":           Delete,
		"Control+DeleteForward":   DeleteWord,
		"Alt+DeleteForward":       DeleteWord,
		"Control+K":               Kill,
		"Control+C":               Copy,
		"Control+X":               Cut,
		"Control+V":               Paste,
		"Shift+Control+V":         PasteHist,
		"Alt+D":                   Duplicate,
		"Control+T":               Transpose,
		"Alt+T":                   TransposeWord,
		"Control+Z":               Undo,
		"Control+Y":               Redo,
		"Shift+Control+Z":         Redo,
		"Control+Alt+I":           Insert,
		"Control+Alt+O":           InsertAfter,
		"Control+=":               ZoomIn,
		"Shift+Control++":         ZoomIn,
		"Control+-":               ZoomOut,
		"Shift+Control+_":         ZoomOut,
		"Shift+Control+P":         Prefs,
		"Control+Alt+P":           Prefs,
		"F5":                      Refresh,
		"Control+L":               Recenter,
		"Control+.":               Complete,
		"Control+,":               Lookup,
		"Alt+S":                   Search,
		"Control+F":               Find,
		"Control+H":               Replace,
		"Control+R":               Replace,
		"Control+J":               Jump,
		"Control+[":               HistPrev,
		"Control+]":               HistNext,
		"F10":                     Menu,
		"Alt+F6":                  WinFocusNext,
		"Control+W":               WinClose,
		"Control+Alt+G":           WinSnapshot,
		"Shift+Control+G":         WinSnapshot,
		"Shift+Control+I":         Inspector,
		"Control+N":               New,
		"Shift+Control+N":         NewAlt1,
		"Control+Alt+N":           NewAlt2,
		"Control+O":               Open,
		"Shift+Control+O":         OpenAlt1,
		"Shift+Alt+O":             OpenAlt2,
		"Control+S":               Save,
		"Shift+Control+S":         SaveAs,
		"Control+Alt+S":           SaveAlt,
		"Shift+Control+W":         CloseAlt1,
		"Control+Alt+W":           CloseAlt2,
	}},
	{"ChromeStd", "Standard chrome-browser and linux-under-chrome bindings", Map{
		"UpArrow":                 MoveUp,
		"Shift+UpArrow":           MoveUp,
		"DownArrow":               MoveDown,
		"Shift+DownArrow":         MoveDown,
		"RightArrow":              MoveRight,
		"Shift+RightArrow":        MoveRight,
		"LeftArrow":               MoveLeft,
		"Shift+LeftArrow":         MoveLeft,
		"PageUp":                  PageUp,
		"Shift+PageUp":            PageUp,
		"Control+UpArrow":         PageUp,
		"PageDown":                PageDown,
		"Shift+PageDown":          PageDown,
		"Control+DownArrow":       PageDown,
		"Home":                    Home,
		"Alt+LeftArrow":           Home,
		"End":                     End,
		"Alt+Home":                DocHome,
		"Shift+Home":              DocHome,
		"Alt+End":                 DocEnd,
		"Shift+End":               DocEnd,
		"Control+RightArrow":      WordRight,
		"Control+LeftArrow":       WordLeft,
		"Alt+RightArrow":          End,
		"Tab":                     FocusNext,
		"Shift+Tab":               FocusPrev,
		"ReturnEnter":             Enter,
		"KeypadEnter":             Enter,
		"Control+A":               SelectAll,
		"Shift+Control+A":         CancelSelect,
		"Control+G":               CancelSelect,
		"Control+Spacebar":        SelectMode, // change input method / keyboard
		"Control+ReturnEnter":     Accept,
		"Escape":                  Abort,
		"DeleteBackspace":         Backspace,
		"Control+DeleteBackspace": BackspaceWord,
		"DeleteForward":           Delete,
		"Control+DeleteForward":   DeleteWord,
		"Alt+DeleteForward":       DeleteWord,
		"Control+K":               Kill,
		"Control+C":               Copy,
		"Control+X":               Cut,
		"Control+V":               Paste,
		"Shift+Control+V":         PasteHist,
		"Alt+D":                   Duplicate,
		"Control+T":               Transpose,
		"Alt+T":                   TransposeWord,
		"Control+Z":               Undo,
		"Control+Y":               Redo,
		"Shift+Control+Z":         Redo,
		"Control+Alt+I":           Insert,
		"Control+Alt+O":           InsertAfter,
		"Control+=":               ZoomIn,
		"Shift+Control++":         ZoomIn,
		"Control+-":               ZoomOut,
		"Shift+Control+_":         ZoomOut,
		"Shift+Control+P":         Prefs,
		"Control+Alt+P":           Prefs,
		"F5":                      Refresh,
		"Control+L":               Recenter,
		"Control+.":               Complete,
		"Control+,":               Lookup,
		"Alt+S":                   Search,
		"Control+F":               Find,
		"Control+H":               Replace,
		"Control+R":               Replace,
		"Control+J":               Jump,
		"Control+[":               HistPrev,
		"Control+]":               HistNext,
		"F10":                     Menu,
		"Alt+F6":                  WinFocusNext,
		"Control+W":               WinClose,
		"Control+Alt+G":           WinSnapshot,
		"Shift+Control+G":         WinSnapshot,
		"Shift+Control+I":         Inspector,
		"Control+N":               New,
		"Shift+Control+N":         NewAlt1,
		"Control+Alt+N":           NewAlt2,
		"Control+O":               Open,
		"Shift+Control+O":         OpenAlt1,
		"Shift+Alt+O":             OpenAlt2,
		"Control+S":               Save,
		"Shift+Control+S":         SaveAs,
		"Control+Alt+S":           SaveAlt,
		"Shift+Control+W":         CloseAlt1,
		"Control+Alt+W":           CloseAlt2,
	}},
}
