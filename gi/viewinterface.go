// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/ki/v2"
)

// ViewInterface is an interface into the View GUI types in the giv subpackage,
// allowing it to be a sub-package with just this narrow set of dependencies
// of gi on giv. The one implementation is in giv/viewinterface.go.
type ViewInterface interface {
	// CallFunc calls the given function in the context of the given widget,
	// popping up a dialog to prompt for any arguments and show the return
	// values of the function. It is a helper function that uses [NewSoloFuncButton]
	// under the hood.
	CallFunc(ctx Widget, fun any)

	// Inspector opens an interactive editor of given Ki tree, at its root
	Inspector(obj ki.Ki)

	// SettingsViewWindow opens a window for editing the user settings
	SettingsViewWindow()
}

// TheViewInterface is the singular implementation of [ViewInterface], defined in the giv package.
var TheViewInterface ViewInterface
