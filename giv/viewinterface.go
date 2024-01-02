// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/ki/v2"
)

// ViewInterface is the singular implementation of [gi.ViewInterface]
type ViewInterface struct{}

func (vi *ViewInterface) CallFunc(ctx gi.Widget, fun any) {
	CallFunc(ctx, fun)
}

func (vi *ViewInterface) Inspector(obj ki.Ki) {
	InspectorWindow(obj)
}

func (vi *ViewInterface) SettingsViewWindow() {
	SettingsViewWindow()
}
