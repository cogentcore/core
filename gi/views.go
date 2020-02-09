// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "github.com/goki/ki/ki"

// ViewIFace is an interface into the View GUI types in giv subpackage,
// allowing it to be a sub-package with just this narrow set of dependencies
// of gi on giv. The one impl is in giv/valueview.go.
type ViewIFace interface {
	// CtxtMenuView configures a popup context menu according to the
	// "CtxtMenu" properties registered on the type for given value element,
	// through the kit.AddType method.  See
	// https://github.com/goki/gi/wiki/Views for full details on formats and
	// options for configuring the menu.  It looks first for "CtxtMenuActive"
	// or "CtxtMenuInactive" depending on inactive flag (which applies to the
	// gui view), so you can have different menus in those cases, and then
	// falls back on "CtxtMenu".  Returns false if there is no context menu
	// defined for this type, or on errors (which are programmer errors sent
	// to log).
	CtxtMenuView(val interface{}, inactive bool, vp *Viewport2D, menu *Menu) bool

	// GoGiEditor opens an interactive editor of given Ki tree, at its root
	GoGiEditor(obj ki.Ki)

	// PrefsView opens an interactive view of given preferences object
	PrefsView(prefs *Preferences)

	// KeyMapsView opens an interactive view of KeyMaps object
	KeyMapsView(maps *KeyMaps)

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
