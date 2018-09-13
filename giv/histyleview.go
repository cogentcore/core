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

// HiStyleName is a highlighting style name
type HiStyleName string

// todo: currently based on https://github.com/alecthomas/chroma styles, but we should
// impl our own structured style obj with a list of categories and
// corresponding colors, once we do the parsing etc

// HiStyles are all the available highlighting styles
var HiStyles = []string{
	"abap",
	"algol",
	"algol_nu",
	"api",
	"arduino",
	"autumn",
	"borland",
	"bw",
	"colorful",
	"dracula",
	"emacs",
	"friendly",
	"fruity",
	"github",
	"igor",
	"lovelace",
	"manni",
	"monokai",
	"monokailight",
	"murphy",
	"native",
	"paraiso-dark",
	"paraiso-light",
	"pastie",
	"perldoc",
	"pygments",
	"rainbow_dash",
	"rrt",
	"solarized-dark",
	"solarized-dark256",
	"solarized-light",
	"swapoff",
	"tango",
	"trac",
	"vim",
	"vs",
	"xcode",
}

////////////////////////////////////////////////////////////////////////////////////////
//  HiStyleValueView

// HiStyleValueView presents an action for displaying an KeyMapName and selecting
// icons from KeyMapChooserDialog
type HiStyleValueView struct {
	ValueViewBase
}

var KiT_HiStyleValueView = kit.Types.AddType(&HiStyleValueView{}, nil)

func (vv *HiStyleValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *HiStyleValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	ac.SetFullReRender()
	ac.SetText(txt)
}

func (vv *HiStyleValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Action)
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_HiStyleValueView).(*HiStyleValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.Viewport, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *HiStyleValueView) HasAction() bool {
	return true
}

func (vv *HiStyleValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	cur := kit.ToString(vv.Value.Interface())
	desc, _ := vv.Tag("desc")
	SliceViewSelectDialog(vp, &HiStyles, cur, DlgOpts{Title: "Select a HiStyle Highlighting Style", Prompt: desc}, nil,
		vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				ddlg, _ := send.(*gi.Dialog)
				si := SliceViewSelectDialogValue(ddlg)
				if si >= 0 {
					hs := HiStyles[si]
					vv.SetValue(hs)
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}
