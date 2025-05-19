// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coresymbols

import (
	"reflect"

	. "cogentcore.org/core/icons"
)

// iconsList is a subset of icons to include in the yaegi symbols.
// It is based on the icons used in the core docs.
var iconsList = map[string]Icon{"Download": Download, "Share": Share, "Send": Send, "Computer": Computer, "Smartphone": Smartphone, "Sort": Sort, "Home": Home, "HomeFill": HomeFill, "DeployedCodeFill": DeployedCodeFill, "Close": Close, "Explore": Explore, "History": History, "Euro": Euro, "OpenInNew": OpenInNew, "Add": Add}

func init() {
	m := map[string]reflect.Value{}
	for name, icon := range iconsList {
		m[name] = reflect.ValueOf(icon)
	}
	Symbols["cogentcore.org/core/icons/icons"] = m
}
