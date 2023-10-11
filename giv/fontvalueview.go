// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/paint"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/laser"
)

////////////////////////////////////////////////////////////////////////////////////////
//  FontValue

// FontValue presents an action for displaying a FontName and selecting
// fonts from FontChooserDialog
type FontValue struct {
	ValueBase
}

func (vv *FontValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *FontValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	ac.SetProp("font-family", txt)
	ac.SetText(txt)
}

func (vv *FontValue) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Button)
	ac.SetProp("border-radius", units.Dp(4))
	ac.OnClick(func(e events.Event) {
		vv.OpenDialog(vv.Widget, nil)
	})
	vv.UpdateWidget()
}

func (vv *FontValue) HasDialog() bool {
	return true
}

func (vv *FontValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	// cur := gi.FontName(laser.ToString(vvv.Value.Interface()))
	desc, _ := vv.Tag("desc")
	FontChooserDialog(ctx, DlgOpts{Title: "Select a Font", Prompt: desc}, func(dlg *gi.Dialog) {
		if dlg.Accepted {
			si := dlg.Data.(int)
			if si >= 0 {
				fi := paint.FontLibrary.FontInfo[si]
				vv.SetValue(fi.Name)
				vv.UpdateWidget()
			}
		}
		if fun != nil {
			fun(dlg)
		}
	}).Run()
}
