// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coresymbols

import (
	"reflect"

	"cogentcore.org/core/icons"
)

// iconsList is a subset of icons to include in the symbols.
// It is based on the icons used in the core docs.
var iconsList = map[string]icons.Icon{"Download": icons.Download, "Share": icons.Share, "Send": icons.Send, "Computer": icons.Computer, "Smartphone": icons.Smartphone, "Sort": icons.Sort, "Home": icons.Home, "HomeFill": icons.HomeFill, "DeployedCodeFill": icons.DeployedCodeFill, "Close": icons.Close, "Explore": icons.Explore, "History": icons.History, "Euro": icons.Euro, "OpenInNew": icons.OpenInNew, "Add": icons.Add}

var _ = register(func() {
	m := map[string]reflect.Value{}
	for name, icon := range iconsList {
		m[name] = reflect.ValueOf(icon)
	}
	add(map[string]map[string]reflect.Value{"cogentcore.org/core/icons": m})
})
