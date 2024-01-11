// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package histyle

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"goki.dev/goki/gi"
	"goki.dev/goki/pi"
)

//go:embed defaults.histys
var defaults []byte

// Styles is a collection of styles
type Styles map[string]*Style

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

// Open hi styles from a JSON-formatted file. You can save and open
// styles to / from files to share, experiment, transfer, etc.
func (hs *Styles) OpenJSON(filename gi.FileName) error { //gti:add
	b, err := os.ReadFile(string(filename))
	if err != nil {
		// PromptDialog(nil, "File Not Found", err.Error(), true, false, nil, nil, nil)
		// slog.Error(err.Error())
		return err
	}
	return json.Unmarshal(b, hs)
}

// Save hi styles to a JSON-formatted file. You can save and open
// styles to / from files to share, experiment, transfer, etc.
func (hs *Styles) SaveJSON(filename gi.FileName) error { //gti:add
	b, err := json.MarshalIndent(hs, "", "  ")
	if err != nil {
		slog.Error(err.Error()) // unlikely
		return err
	}
	err = os.WriteFile(string(filename), b, 0644)
	if err != nil {
		// PromptDialog(nil, "Could not Save to File", err.Error(), true, false, nil, nil, nil)
		slog.Error(err.Error())
	}
	return err
}

// PrefsStylesFileName is the name of the preferences file in App prefs
// directory for saving / loading the custom styles
var PrefsStylesFileName = "hi_styles.json"

// StylesChanged is used for gui updating while editing
var StylesChanged = false

// OpenPrefs opens Styles from GoGi standard prefs directory, using PrefsStylesFileName
func (hs *Styles) OpenPrefs() error {
	pdir := gi.GokiDataDir()
	pnm := filepath.Join(pdir, PrefsStylesFileName)
	StylesChanged = false
	return hs.OpenJSON(gi.FileName(pnm))
}

// SavePrefs saves Styles to GoGi standard prefs directory, using PrefsStylesFileName
func (hs *Styles) SavePrefs() error {
	pdir := gi.GokiDataDir()
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
// These are encoded as an embed from defaults.histys
func (hs *Styles) OpenDefaults() error {
	err := json.Unmarshal(defaults, hs)
	if err != nil {
		slog.Error(err.Error())
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
	View(&StdStyles)
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

	for _, s := range AvailStyles {
		for _, se := range *s {
			se.Norm()
		}
	}
}
