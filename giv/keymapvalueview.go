// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  KeyMapValueView

// KeyMapValueView presents an action for displaying an KeyMapName and selecting
// icons from KeyMapChooserDialog
type KeyMapValueView struct {
	ValueViewBase
}

var KiT_KeyMapValueView = kit.Types.AddType(&KeyMapValueView{}, nil)

func (vv *KeyMapValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *KeyMapValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	ac.SetFullReRender()
	ac.SetText(txt)
}

func (vv *KeyMapValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Action)
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_KeyMapValueView).(*KeyMapValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.Viewport, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *KeyMapValueView) HasAction() bool {
	return true
}

func (vv *KeyMapValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	cur := kit.ToString(vv.Value.Interface())
	desc, _ := vv.Tag("desc")
	SliceViewSelectDialog(vp, &gi.StdKeyMapNames, cur, DlgOpts{Title: "Select a Standard KeyMap", Prompt: desc}, nil,
		vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			ddlg, _ := send.(*gi.Dialog)
			if sig == int64(gi.DialogAccepted) {
				si := SliceViewSelectDialogValue(ddlg)
				if si >= 0 {
					km := gi.StdKeyMapNames[si]
					vv.SetValue(km)
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}
