// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/ki"
)

// Variables for functions the giv package, allowing for it to be a separate package
// with just this narrow set of dependencies of gi on giv.

// TODO(kai): should probably get rid of gi.CallFunc

// CallFunc calls the given function in the context of the given widget,
// popping up a dialog to prompt for any arguments and show the return
// values of the function. It is set to [giv.CallFunc] if giv is imported;
// otherwise, it is nil.
var CallFunc func(ctx Widget, fun any)

// InspectorWindow opens an interactive editor of the given Ki tree at its root.
// It is set to [giv.InspectorWindow] if giv is imported; otherwise; it is nil.
var InspectorWindow func(obj ki.Ki)

// SettingsWindow opens a window for editing the user settings.
// It is set to [giv.SettingsWindow] if giv is imported; otherwise; it is nil.
var SettingsWindow func()
