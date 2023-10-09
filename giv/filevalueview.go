// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gti"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

////////////////////////////////////////////////////////////////////////////////////////
//  FileValueView

// FileValueView presents an action for displaying a FileName and selecting
// icons from FileChooserDialog
type FileValueView struct {
	ValueViewBase
}

func (vv *FileValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ActionType
	return vv.WidgetTyp
}

func (vv *FileValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(click to open file chooser)"
	}
	ac.SetText(txt)
}

func (vv *FileValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Action)
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(TypeFileValueView).(*FileValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.OpenDialog(ac.Sc, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *FileValueView) HasAction() bool {
	return true
}

func (vv *FileValueView) OpenDialog(vp *gi.Scene, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	ext, _ := vv.Tag("ext")
	desc, _ := vv.Tag("desc")
	FileViewDialog(vp, cur, ext, DlgOpts{Title: vv.Name(), Prompt: desc}, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				dlg, _ := send.Embed(gi.TypeDialog).(*gi.Dialog)
				fn := FileViewDialogValue(dlg)
				vv.SetValue(fn)
				vv.UpdateWidget()
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}
