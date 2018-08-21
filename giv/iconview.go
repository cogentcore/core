// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"
	"strings"

	"github.com/goki/gi"
	"github.com/goki/ki"
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
	cb := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	if gi.IconName(txt).IsNil() {
		cb.SetIcon("blank")
	} else {
		cb.SetIcon(txt)
	}
	sntag := vv.ViewFieldTag("view")
	if strings.Contains(sntag, "show-name") {
		if txt == "" {
			txt = "none"
		}
		cb.SetText(txt)
	}
}

func (vv *IconValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg

	cb := vv.Widget.(*gi.Action)
	vv.UpdateWidget()

	cb.ActionSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_IconValueView).(*IconValueView)
		if !vvv.IsInactive() {
			cur := gi.IconName(kit.ToString(vvv.Value.Interface()))
			IconChooserDialog(cb.Viewport, cur, "Select an Icon", "", nil, vv.This,
				func(recv, send ki.Ki, sig int64, data interface{}) {
					sv, _ := send.(*SliceView)
					si := sv.SelectedIdx
					if si >= 0 {
						cbb := vvv.Widget.(*gi.Action)
						ic := gi.CurIconList[si]
						vvv.SetValue(ic)
						cbb.SetIcon(string(ic))
					}
				}, nil)
		}
	})
}
