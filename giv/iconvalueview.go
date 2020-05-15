// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  IconValueView

// IconValueView presents an action for displaying an IconName and selecting
// icons from IconChooserDialog
type IconValueView struct {
	ValueViewBase
}

var KiT_IconValueView = kit.Types.AddType(&IconValueView{}, nil)

func (vv *IconValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *IconValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	if gi.IconName(txt).IsNil() {
		ac.SetIcon("blank")
	} else {
		ac.SetIcon(txt)
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
	ac.SetProp("border-radius", units.NewPx(4))
	ac.SetProp("padding", 0)
	ac.SetProp("margin", 0)
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_IconValueView).(*IconValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.ViewportSafe(), nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *IconValueView) HasAction() bool {
	return true
}

func (vv *IconValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	cur := gi.IconName(kit.ToString(vv.Value.Interface()))
	desc, _ := vv.Tag("desc")
	IconChooserDialog(vp, cur, DlgOpts{Title: "Select an Icon", Prompt: desc},
		vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
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
