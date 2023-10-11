// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/laser"
)

////////////////////////////////////////////////////////////////////////////////////////
//  FileValue

// FileValue presents an action for displaying a FileName and selecting
// icons from FileChooserDialog
type FileValue struct {
	ValueBase
}

func (vv *FileValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *FileValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(click to open file chooser)"
	}
	ac.SetText(txt)
}

func (vv *FileValue) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Button)
	ac.OnClick(func(e events.Event) {
		ac := vv.Widget.(*gi.Button)
		vv.OpenDialog(ac, nil)
	})
	vv.UpdateWidget()
}

func (vv *FileValue) HasDialog() bool {
	return true
}

func (vv *FileValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	ext, _ := vv.Tag("ext")
	desc, _ := vv.Tag("desc")
	FileViewDialog(ctx, DlgOpts{Title: vv.Name(), Prompt: desc}, cur, ext, nil, func(dlg *gi.Dialog) {
		if dlg.Accepted {
			fn := dlg.Data.(string)
			vv.SetValue(fn)
			vv.UpdateWidget()
		}
		if fun != nil {
			fun(dlg)
		}
	}).Run()
}
