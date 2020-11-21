// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

// Prefs defines the interface to preferences for style-relevant prefs
type Prefs interface {
	// PrefColor returns preference color of given name
	// std names are: font, background, shadow, border, control, icon, select, highlight, link
	// nil if not found
	PrefColor(name string) *Color

	// PrefFontFamily returns the default FontFamily
	PrefFontFamily() string
}

// ThePrefs is the prefs object to use to get preferences
var ThePrefs Prefs
