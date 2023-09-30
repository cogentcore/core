// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

// Themes are the different possible themes that a user can select in their preferences.
type Themes int32 //enums:enum -trimprefix Theme

const (
	// ThemeAuto indicates to use the theme specified by the operating system
	ThemeAuto Themes = iota
	// ThemeLight indicates to use a light theme
	ThemeLight
	// ThemeDark indicates to use a dark theme
	ThemeDark
)
