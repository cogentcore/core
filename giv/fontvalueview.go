// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/girl"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

////////////////////////////////////////////////////////////////////////////////////////
//  FontValueView

// FontValueView presents an action for displaying a FontName and selecting
// fonts from FontChooserDialog
type FontValueView struct {
	ValueViewBase
}

func (vv *FontValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.ActionType
	return vv.WidgetTyp
}

func (vv *FontValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := laser.ToString(vv.Value.Interface())
	ac.SetProp("font-family", txt)
	ac.SetText(txt)
}

func (vv *FontValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Action)
	ac.SetProp("border-radius", units.Px(4))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(TypeFontValueView).(*FontValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.Vp, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *FontValueView) HasAction() bool {
	return true
}

func (vv *FontValueView) Activate(vp *gi.Viewport, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	// cur := gi.FontName(laser.ToString(vvv.Value.Interface()))
	desc, _ := vv.Tag("desc")
	FontChooserDialog(vp, DlgOpts{Title: "Select a Font", Prompt: desc},
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.TypeDialog).(*gi.Dialog)
				si := TableViewSelectDialogValue(ddlg)
				if si >= 0 {
					fi := girl.FontLibrary.FontInfo[si]
					vv.SetValue(fi.Name)
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}
