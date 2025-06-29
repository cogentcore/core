// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package icons provides Material Design Symbols as SVG icon variables.
package icons

import (
	_ "embed"
	"sync"

	"golang.org/x/exp/maps"
)

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

	// used is a map containing all icons that have been used.
	// It is added to by [cogentcore.org/core/core.Icon].
	used = map[Icon]struct{}{}

	usedMu sync.Mutex
)

// IsSet returns whether the icon is set to a value other than "" or [None].
func (i Icon) IsSet() bool {
	return i != "" && i != None
}

// AddUsed adds given icon to the list of icons that have been used
// (under mutex lock).
func AddUsed(i Icon) {
	usedMu.Lock()
	used[i] = struct{}{}
	usedMu.Unlock()
}

// Used returns a list of icons that have been used so far in the app.
// This list is in indeterminate order.
func Used() []Icon {
	usedMu.Lock()
	defer usedMu.Unlock()
	return maps.Keys(used)
}
