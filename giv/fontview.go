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
//  FontValueView

// FontValueView presents an action for displaying a FontName and selecting
// fonts from FontChooserDialog
type FontValueView struct {
	ValueViewBase
}

var KiT_FontValueView = kit.Types.AddType(&FontValueView{}, nil)

func (vv *FontValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *FontValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	ac.SetFullReRender()
	txt := kit.ToString(vv.Value.Interface())
	ac.SetProp("font-family", txt)
	ac.SetText(txt)
}

func (vv *FontValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Action)
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_FontValueView).(*FontValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.Viewport)
	})
	vv.UpdateWidget()
}

func (vv *FontValueView) HasAction() bool {
	return true
}

func (vv *FontValueView) Activate(vp *gi.Viewport2D) {
	if vv.IsInactive() {
		return
	}
	// cur := gi.FontName(kit.ToString(vvv.Value.Interface()))
	FontChooserDialog(vp, "Select a Font", "", nil, vv.This, nil,
		func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
				si := TableViewSelectDialogValue(ddlg)
				if si >= 0 {
					fi := gi.FontLibrary.FontInfo[si]
					vv.SetValue(fi.Name)
					vv.UpdateWidget()
				}
			}
		})
}
