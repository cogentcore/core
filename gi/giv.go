// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"cogentcore.org/core/ki"
)

// Variables for functions the giv package, allowing for it to be a separate package
// with just this narrow set of dependencies of gi on giv.

// InspectorWindow opens an interactive editor of the given Ki tree at its root.
// It is set to [giv.InspectorWindow] if giv is imported; otherwise; it is nil.
var InspectorWindow func(obj ki.Node)

// SettingsWindow opens a window for editing the user settings.
// It is set to [giv.SettingsWindow] if giv is imported; otherwise; it is nil.
var SettingsWindow func()
