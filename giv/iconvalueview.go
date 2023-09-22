// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"
	"strings"

	"goki.dev/gi/v2/gi"
	"goki.dev/gicons"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

////////////////////////////////////////////////////////////////////////////////////////
//  IconValueView

// IconValueView presents an action for displaying an IconName and selecting
// icons from IconChooserDialog
type IconValueView struct {
	ValueViewBase
}

func (vv *IconValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.ActionType
	return vv.WidgetTyp
}

func (vv *IconValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := laser.ToString(vv.Value.Interface())
	if gicons.Icon(txt).IsNil() {
		ac.SetIcon("blank")
	} else {
		ac.SetIcon(gicons.Icon(txt))
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

func (vv *IconValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Action)
	ac.SetProp("border-radius", units.Px(4))
	ac.SetProp("padding", 0)
	ac.SetProp("margin", 0)
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(IconTypeValueView).(*IconValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.Vp, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *IconValueView) HasAction() bool {
	return true
}

func (vv *IconValueView) Activate(vp *gi.Viewport, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	cur := gicons.Icon(laser.ToString(vv.Value.Interface()))
	desc, _ := vv.Tag("desc")
	IconChooserDialog(vp, cur, DlgOpts{Title: "Select an Icon", Prompt: desc},
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.TypeDialog).(*gi.Dialog)
				si := SliceViewSelectDialogValue(ddlg)
				if si >= 0 {
					ic := gi.CurIconList[si]
					vv.SetValue(ic)
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}
