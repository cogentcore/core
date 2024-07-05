// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package icons provides Material Design Symbols as SVG icon variables.
package icons

import _ "embed"

//go:generate go run gen.go

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

// All is a list of all icons compiled into the app.
// It does not contain unused icons, since those are
// not compiled. [Register] adds to it in generated
// icongen code.
var All []Icon

// Register adds the given [Icon] to [All] and then
// returns the same icon. This should only be used
// in generated icongen code.
func Register(i Icon) Icon {
	All = append(All, i)
	return i
}

// DefaultAppIcon is the default icon used for apps during packaging and in the app
// if no icon is specified in the icon.svg file. It defaults to a Google Blue version
// of [Toolbar].
const DefaultAppIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 -960 960 960"><path fill="#4285f4" d="M180-120q-24 0-42-18t-18-42v-600q0-24 18-42t42-18h600q24 0 42 18t18 42v600q0 24-18 42t-42 18H180Zm0-513h600v-147H180v147Zm600 60H180v393h600v-393Zm-600-60v60-60Zm0 0v-147 147Zm0 60v393-393Z"/></svg>`
