// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/tree"
)

// Variables for functions the views package, allowing for it to be a separate package
// with just this narrow set of dependencies of core on views.

// InspectorWindow opens an interactive editor of the given Ki tree at its root.
// It is set to [views.InspectorWindow] if views is imported; otherwise; it is nil.
var InspectorWindow func(obj tree.Node)

// SettingsWindow opens a window for editing the user settings.
// It is set to [views.SettingsWindow] if views is imported; otherwise; it is nil.
var SettingsWindow func()
