// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package histyle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/pi"
)

// Styles is a collection of styles
type Styles map[string]*Style

var KiT_Styles = kit.Types.AddType(&Styles{}, StylesProps)

// StdStyles are the styles from chroma package
var StdStyles Styles

// CustomStyles are user's special styles
var CustomStyles = Styles{}

// AvailStyles are all highlighting styles
var AvailStyles Styles

// StyleDefault is the default highlighting style name -- can set this to whatever you want
var StyleDefault = gi.HiStyleName("emacs")

// StyleNames are all the names of all the available highlighting styles
var StyleNames []string

// AvailStyle returns a style by name from the AvailStyles list -- if not found
// default is used as a fallback
func AvailStyle(nm gi.HiStyleName) *Style {
	if AvailStyles == nil {
		Init()
	}
	if st, ok := AvailStyles[string(nm)]; ok {
		return st
	}
	return AvailStyles[string(StyleDefault)]
}

// Add adds a new style to the list
func (hs *Styles) Add() *Style {
	hse := &Style{}
	nm := fmt.Sprintf("NewStyle_%v", len(*hs))
	(*hs)[nm] = hse
	return hse
}

// CopyFrom copies styles from another collection
func (hs *Styles) CopyFrom(os Styles) {
	if *hs == nil {
		*hs = make(Styles, len(os))
	}
	for nm, cse := range os {
		(*hs)[nm] = cse
	}
}

// MergeAvailStyles updates AvailStyles as combination of std and custom styles
func MergeAvailStyles() {
	AvailStyles = make(Styles, len(CustomStyles)+len(StdStyles))
	AvailStyles.CopyFrom(StdStyles)
	AvailStyles.CopyFrom(CustomStyles)
	StyleNames = AvailStyles.Names()
}

// Open hi styles from a JSON-formatted file.
func (hs *Styles) OpenJSON(filename gi.FileName) error {
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		// PromptDialog(nil, "File Not Found", err.Error(), true, false, nil, nil, nil)
		// log.Println(err)
		return err
	}
	return json.Unmarshal(b, hs)
}

// Save hi styles to a JSON-formatted file.
func (hs *Styles) SaveJSON(filename gi.FileName) error {
	b, err := json.MarshalIndent(hs, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(string(filename), b, 0644)
	if err != nil {
		// PromptDialog(nil, "Could not Save to File", err.Error(), true, false, nil, nil, nil)
		log.Println(err)
	}
	return err
}

// PrefsStylesFileName is the name of the preferences file in App prefs
// directory for saving / loading the custom styles
var PrefsStylesFileName = "hi_styles.json"

// StylesChanged is used for gui updating while editing
var StylesChanged = false

// OpenPrefs opens Styles from App standard prefs directory, using PrefsStylesFileName
func (hs *Styles) OpenPrefs() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsStylesFileName)
	StylesChanged = false
	return hs.OpenJSON(gi.FileName(pnm))
}

// SavePrefs saves Styles to App standard prefs directory, using PrefsStylesFileName
func (hs *Styles) SavePrefs() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsStylesFileName)
	StylesChanged = false
	MergeAvailStyles()
	return hs.SaveJSON(gi.FileName(pnm))
}

// SaveAll saves all styles individually to chosen directory
func (hs *Styles) SaveAll(dir gi.FileName) {
	for nm, st := range *hs {
		fnm := filepath.Join(string(dir), nm+".histy")
		st.SaveJSON(gi.FileName(fnm))
	}
}

// OpenDefaults opens the default highlighting styles (from chroma originally)
// These are encoded as an asset from defaults.histys
func (hs *Styles) OpenDefaults() error {
	// path, err := dirs.GoSrcDir("github.com/goki/gi/histyle")
	// if err != nil {
	// 	log.Println(err)
	// 	return err
	// }
	// dfnm := filepath.Join(path, "defaults.histys")
	// rval := hs.OpenJSON(gi.FileName(dfnm))

	defb, err := Asset("defaults.histys")
	if err != nil {
		log.Println(err)
		return err
	}
	err = json.Unmarshal(defb, hs)
	if err != nil {
		log.Println(err)
		return err
	}
	return err
}

// Names outputs names of styles in collection
func (hs *Styles) Names() []string {
	nms := make([]string, len(*hs))
	idx := 0
	for nm := range *hs {
		nms[idx] = nm
		idx++
	}
	sort.StringSlice(nms).Sort()
	return nms
}

// ViewStd shows the standard styles that are compiled into the program via
// chroma package
func (hs *Styles) ViewStd() {
	gi.TheViewIFace.HiStylesView(true)
}

// Init must be called to initialize the hi styles -- post startup
// so chroma stuff is all in place, and loads custom styles
func Init() {
	pi.LangSupport.OpenStd()
	StdStyles.OpenDefaults()
	CustomStyles.OpenPrefs()
	if len(CustomStyles) == 0 {
		cs := &Style{}
		cs.CopyFrom(StdStyles[string(StyleDefault)])
		CustomStyles["custom-sample"] = cs
	}
	MergeAvailStyles()
}

// StylesProps define the ToolBar and MenuBar for view
var StylesProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenPrefs", ki.Props{}},
			{"SavePrefs", ki.Props{
				"shortcut": gi.KeyFunMenuSave,
				"updtfunc": func(sti interface{}, act *gi.Action) {
					act.SetActiveStateUpdt(StylesChanged && sti.(*Styles) == &CustomStyles)
				},
			}},
			{"sep-file", ki.BlankProp{}},
			{"OpenJSON", ki.Props{
				"label":    "Open...",
				"desc":     "You can save and open styles to / from files to share, experiment, transfer, etc",
				"shortcut": gi.KeyFunMenuOpen,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"SaveJSON", ki.Props{
				"label":    "Save As...",
				"desc":     "You can save and open styles to / from files to share, experiment, transfer, etc",
				"shortcut": gi.KeyFunMenuSaveAs,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"SaveAll", ki.Props{
				"label": "Save All...",
				"desc":  "Saves each style individually to selected directory (be sure to select a dir only!)",
				"Args": ki.PropSlice{
					{"Dir Name", ki.Props{}},
				},
			}},
		}},
		{"Edit", "Copy Cut Paste Dupe"},
		{"Window", "Windows"},
	},
	"ToolBar": ki.PropSlice{
		{"Add", ki.Props{ // note: overrides default Add
			"desc": "Add a new style to the list.",
			"icon": "plus",
			"updtfunc": func(sti interface{}, act *gi.Action) {
				act.SetActiveStateUpdt(sti.(*Styles) == &CustomStyles)
			},
		}},
		{"SavePrefs", ki.Props{
			"desc": "saves styles to app prefs directory, in file hi_styles.json, which will be loaded automatically at startup into your CustomStyles.",
			"icon": "file-save",
			"updtfunc": func(sti interface{}, act *gi.Action) {
				act.SetActiveStateUpdt(StylesChanged && sti.(*Styles) == &CustomStyles)
			},
		}},
		{"sep-file", ki.BlankProp{}},
		{"OpenJSON", ki.Props{
			"label": "Open from file",
			"icon":  "file-open",
			"desc":  "You can save and open styles to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"SaveJSON", ki.Props{
			"label": "Save to file",
			"icon":  "file-save",
			"desc":  "You can save and open styles to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"sep-std", ki.BlankProp{}},
		{"ViewStd", ki.Props{
			"desc":    `Shows the standard styles that are compiled into the program (from <a href="https://github.com/alecthomas/chroma">github.com/alecthomas/chroma</a>).  Save a style from there and load it into custom as a starting point for creating a variant of an existing style.`,
			"confirm": true,
			"updtfunc": func(sti interface{}, act *gi.Action) {
				act.SetActiveStateUpdt(sti.(*Styles) != &StdStyles)
			},
		}},
	},
}
