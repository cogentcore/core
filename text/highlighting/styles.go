// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/system"
	"cogentcore.org/core/text/parse"
)

// DefaultStyle is the initial default style.
var DefaultStyle = HighlightingName("emacs")

// Styles is a collection of styles
type Styles map[string]*Style

var (
	//go:embed defaults.highlighting
	defaults []byte

	// StandardStyles are the styles from chroma package
	StandardStyles Styles

	// CustomStyles are user's special styles
	CustomStyles = Styles{}

	// AvailableStyles are all highlighting styles
	AvailableStyles Styles

	// StyleDefault is the default highlighting style name
	StyleDefault = HighlightingName("emacs")

	// StyleNames are all the names of all the available highlighting styles
	StyleNames []string

	// SettingsStylesFilename is the name of the preferences file in App data
	// directory for saving / loading the custom styles
	SettingsStylesFilename = "highlighting.json"

	// StylesChanged is used for gui updating while editing
	StylesChanged = false
)

// UpdateFromTheme normalizes the colors of all style entry such that they have consistent
// chromas and tones that guarantee sufficient text contrast in accordance with the color theme.
func UpdateFromTheme() {
	for _, s := range AvailableStyles {
		for _, se := range *s {
			se.UpdateFromTheme()
		}
	}
}

// AvailableStyle returns a style by name from the AvailStyles list -- if not found
// default is used as a fallback
func AvailableStyle(nm HighlightingName) *Style {
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
func (hs *Styles) OpenJSON(filename fsx.Filename) error { //types:add
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
func (hs *Styles) SaveJSON(filename fsx.Filename) error { //types:add
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

// OpenSettings opens Styles from Cogent Core standard prefs directory, using SettingsStylesFilename
func (hs *Styles) OpenSettings() error {
	pdir := system.TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, SettingsStylesFilename)
	StylesChanged = false
	return hs.OpenJSON(fsx.Filename(pnm))
}

// SaveSettings saves Styles to Cogent Core standard prefs directory, using SettingsStylesFilename
func (hs *Styles) SaveSettings() error {
	pdir := system.TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, SettingsStylesFilename)
	StylesChanged = false
	MergeAvailStyles()
	return hs.SaveJSON(fsx.Filename(pnm))
}

// SaveAll saves all styles individually to chosen directory
func (hs *Styles) SaveAll(dir fsx.Filename) {
	for nm, st := range *hs {
		fnm := filepath.Join(string(dir), nm+".highlighting")
		st.SaveJSON(fsx.Filename(fnm))
	}
}

// OpenDefaults opens the default highlighting styles (from chroma originally)
// These are encoded as an embed from defaults.highlighting
func (hs *Styles) OpenDefaults() error {
	err := json.Unmarshal(defaults, hs)
	if err != nil {
		return errors.Log(err)
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
	slices.Sort(nms)
	return nms
}

// Init must be called to initialize the hi styles -- post startup
// so chroma stuff is all in place, and loads custom styles
func Init() {
	parse.LanguageSupport.OpenStandard()
	StandardStyles.OpenDefaults()
	CustomStyles.OpenSettings()
	if len(CustomStyles) == 0 {
		cs := &Style{}
		cs.CopyFrom(StandardStyles[string(StyleDefault)])
		CustomStyles["custom-sample"] = cs
	}
	MergeAvailStyles()
	UpdateFromTheme()
}
