// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

// Prefer defines the interface to preferences for style-relevant prefs
type Prefer interface {
	// PrefFontFamily returns the default FontFamily
	PrefFontFamily() string
}

// ThePrefs is the prefs object to use to get preferences.
var ThePrefs Prefer

// Prefs provides a basic implementation of Prefer interface
type Prefs struct {

	// font family name
	FontFamily string `desc:"font family name"`
}

func (pf *Prefs) Defaults() {
	pf.FontFamily = "Go" // TODO(kai): change this to Roboto
}

// PrefFontFamily returns the default FontFamily
func (pf *Prefs) PrefFontFamily() string {
	return pf.FontFamily
}
