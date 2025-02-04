// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package icons provides Material Design Symbols as SVG icon variables.
package icons

import _ "embed"

//go:generate core generate -icons svg

// Icon represents the SVG data of an icon. It can be
// set to "" or [None] to indicate that no icon should be used.
type Icon string

var (
	// None is an icon that indicates to not use an icon.
	// It completely prevents the rendering of an icon,
	// whereas [Blank] renders a blank icon.
	None Icon = "none"

	// Blank is a blank icon that can be used as a
	// placeholder when no other icon is appropriate.
	// It still renders an icon, just a blank one,
	// whereas [None] indicates to not render one at all.
	//
	//go:embed svg/blank.svg
	Blank Icon
)

// IsSet returns whether the icon is set to a value other than "" or [None].
func (i Icon) IsSet() bool {
	return i != "" && i != None
}

// Used is a map containing all icons that have been used.
// It is added to by [cogentcore.org/core/core.Icon].
var Used = map[Icon]struct{}{}
