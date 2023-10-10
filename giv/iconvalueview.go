// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"strings"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/laser"
)

////////////////////////////////////////////////////////////////////////////////////////
//  IconValueView

// IconValueView presents an action for displaying an IconName and selecting
// icons from IconChooserDialog
type IconValueView struct {
	ValueViewBase
}

func (vv *IconValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *IconValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if icons.Icon(txt).IsNil() {
		ac.SetIcon("blank")
	} else {
		ac.SetIcon(icons.Icon(txt))
	}
	if sntag, ok := vv.Tag("view"); ok {
		if strings.Contains(sntag, "show-name") {
			if txt == "" {
				txt = "none"
			}
			ac.SetText(txt)
		}
	}
}

func (vv *IconValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Button)
	ac.SetProp("border-radius", units.Dp(4))
	ac.SetProp("padding", 0)
	ac.SetProp("margin", 0)
	ac.OnClick(func(e events.Event) {
		vv.OpenDialog(ac, nil)
	})
	vv.UpdateWidget()
}

func (vv *IconValueView) HasDialog() bool {
	return true
}

func (vv *IconValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	cur := icons.Icon(laser.ToString(vv.Value.Interface()))
	desc, _ := vv.Tag("desc")
	IconChooserDialog(ctx, DlgOpts{Title: "Select an Icon", Prompt: desc}, cur, func(dlg *gi.Dialog) {
		if dlg.Accepted {
			si := SliceViewSelectDialogValue(dlg)
			if si >= 0 {
				ic := gi.CurIconList[si]
				vv.SetValue(ic)
				vv.UpdateWidget()
			}
		}
		if fun != nil {
			fun(dlg)
		}
	})
}
