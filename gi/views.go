// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/gi/v2/keyfun"
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
	PrefsView(prefs *Preferences)

	// KeyMapsView opens an interactive view of KeyMaps object
	KeyMapsView(maps *keyfun.Maps)

	// PrefsDetView opens an interactive view of given detailed preferences object
	PrefsDetView(prefs *PrefsDetailed)

	// HiStylesView opens an interactive view of custom or std highlighting styles.
	HiStylesView(std bool)

	// SetHiStyleDefault sets the current default histyle.StyleDefault
	SetHiStyleDefault(hsty HiStyleName)

	// HiStyleInit initializes the histyle package -- called during overall gi init.
	HiStyleInit()

	// PrefsDetDefaults gets current detailed prefs values as defaults
	PrefsDetDefaults(prefs *PrefsDetailed)

	// PrefsDetApply applies detailed preferences within giv scope
	PrefsDetApply(prefs *PrefsDetailed)

	// PrefsDbgView opens an interactive view of given debugging preferences object
	PrefsDbgView(prefs *PrefsDebug)
}

// TheViewIFace is the implementation of the interface, defined in giv package
var TheViewIFace ViewIFace
