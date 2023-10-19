// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import "goki.dev/gi/v2/gi"

// Toolbarer is an interface that types can satisfy to add a toolbar when they
// are displayed in the GUI. In the Toolbar method, types typically add [FuncButton]
// and [gi.Separator] objects to the toolbar that they are passed, although they can
// do anything they want. [ToolbarView] checks for implementation of this interface.
type Toolbarer interface {
	Toolbar(tb *gi.Toolbar)
}

// ToolbarView calls the Toolbar function of the given value on the given toolbar,
// if the given value is implements the [Toolbarer] interface. Otherwise, it does
// nothing. It returns whether the given value implements that interface.
func ToolbarView(val any, tb *gi.Toolbar) bool {
	tbr, ok := val.(Toolbarer)
	if !ok {
		return false
	}
	tbr.Toolbar(tb)
	return true
}
