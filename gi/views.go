// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/ki/v2"
)

// ViewIFace is an interface into the View GUI types in giv subpackage,
// allowing it to be a sub-package with just this narrow set of dependencies
// of gi on giv. The one impl is in giv/valueview.go.
type ViewIFace interface {
	// CallFunc calls the given function in the context of the given widget,
	// popping up a dialog to prompt for any arguments and show the return
	// values of the function. It is a helper function that uses [NewSoloFuncButton]
	// under the hood.
	CallFunc(ctx Widget, fun any)

	// Inspector opens an interactive editor of given Ki tree, at its root
	Inspector(obj ki.Ki)

	// PrefsView opens an interactive view of given preferences object
	PrefsView(prefs *AppearanceSettingsData)

	// TODO(kai): figure out a better way to structure histyle view things

	// HiStylesView opens an interactive view of custom or std highlighting styles.
	HiStylesView(std bool)

	// SetHiStyleDefault sets the current default histyle.StyleDefault
	SetHiStyleDefault(hsty HiStyleName)

	// HiStyleInit initializes the histyle package -- called during overall gi init.
	HiStyleInit()
}

// TheViewIFace is the implementation of the interface, defined in giv package
var TheViewIFace ViewIFace
