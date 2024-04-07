// Copyright (c) 2018, Cogent Core. All rights reserved.
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

	"cogentcore.org/core/gi"
	"cogentcore.org/core/pi"
)

//go:embed defaults.histys
var defaults []byte

// Styles is a collection of styles
type Styles map[string]*Style

// StandardStyles are the styles from chroma package
var StandardStyles Styles

// CustomStyles are user's special styles
var CustomStyles = Styles{}

// AvailableStyles are all highlighting styles
var AvailableStyles Styles

// StyleDefault is the default highlighting style name -- can set this to whatever you want
var StyleDefault = gi.HiStyleName("emacs")

// StyleNames are all the names of all the available highlighting styles
var StyleNames []string

// AvailableStyle returns a style by name from the AvailStyles list -- if not found
// default is used as a fallback
func AvailableStyle(nm gi.HiStyleName) *Style {
	if AvailableStyles == nil {
		Init()
	}
	if st, ok := AvailableStyles[string(nm)]; ok {
		return st
	}
	return AvailableStyles[string(StyleDefault)]
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
	AvailableStyles = make(Styles, len(CustomStyles)+len(StandardStyles))
	AvailableStyles.CopyFrom(StandardStyles)
	AvailableStyles.CopyFrom(CustomStyles)
	StyleNames = AvailableStyles.Names()
}

// Open hi styles from a JSON-formatted file. You can save and open
// styles to / from files to share, experiment, transfer, etc.
func (hs *Styles) OpenJSON(filename gi.Filename) error { //gti:add
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
func (hs *Styles) SaveJSON(filename gi.Filename) error { //gti:add
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

// SettingsStylesFilename is the name of the preferences file in App prefs
// directory for saving / loading the custom styles
var SettingsStylesFilename = "hi_styles.json"

// StylesChanged is used for gui updating while editing
var StylesChanged = false

// OpenSettings opens Styles from Cogent Core standard prefs directory, using SettingsStylesFilename
func (hs *Styles) OpenSettings() error {
	pdir := gi.TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, SettingsStylesFilename)
	StylesChanged = false
	return hs.OpenJSON(gi.Filename(pnm))
}

// SaveSettings saves Styles to Cogent Core standard prefs directory, using SettingsStylesFilename
func (hs *Styles) SaveSettings() error {
	pdir := gi.TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, SettingsStylesFilename)
	StylesChanged = false
	MergeAvailStyles()
	return hs.SaveJSON(gi.Filename(pnm))
}

// SaveAll saves all styles individually to chosen directory
func (hs *Styles) SaveAll(dir gi.Filename) {
	for nm, st := range *hs {
		fnm := filepath.Join(string(dir), nm+".histy")
		st.SaveJSON(gi.Filename(fnm))
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

// ViewStandard shows the standard styles that are compiled into the program via
// chroma package
func (hs *Styles) ViewStandard() {
	View(&StandardStyles)
}

// Init must be called to initialize the hi styles -- post startup
// so chroma stuff is all in place, and loads custom styles
func Init() {
	pi.LangSupport.OpenStandard()
	StandardStyles.OpenDefaults()
	CustomStyles.OpenSettings()
	if len(CustomStyles) == 0 {
		cs := &Style{}
		cs.CopyFrom(StandardStyles[string(StyleDefault)])
		CustomStyles["custom-sample"] = cs
	}
	MergeAvailStyles()

	for _, s := range AvailableStyles {
		for _, se := range *s {
			se.Norm()
		}
	}
}
