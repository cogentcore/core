// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor/histyle"
	"goki.dev/ki/v2"
)

// ViewInterface is the singular implementation of [gi.ViewInterface]
type ViewInterface struct{}

func (vi *ViewInterface) CallFunc(ctx gi.Widget, fun any) {
	CallFunc(ctx, fun)
}

func (vi *ViewInterface) Inspector(obj ki.Ki) {
	InspectorDialog(obj)
}

func (vi *ViewInterface) SettingsViewWindow() {
	SettingsViewWindow()
}

func (vi *ViewInterface) HiStylesView(std bool) {
	if std {
		HiStylesView(&histyle.StdStyles)
	} else {
		HiStylesView(&histyle.CustomStyles)
	}
}

func (vi *ViewInterface) HiStyleInit() {
	histyle.Init()
}

func (vi *ViewInterface) SetHiStyleDefault(hsty gi.HiStyleName) {
	histyle.StyleDefault = hsty
}
