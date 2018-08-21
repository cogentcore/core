// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"github.com/goki/gi"
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
	cb := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	cb.SetProp("font-family", txt)
	cb.SetText(txt)
}

func (vv *FontValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg

	cb := vv.Widget.(*gi.Action)
	vv.UpdateWidget()

	cb.ActionSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_FontValueView).(*FontValueView)
		if !vvv.IsInactive() {
			// cur := gi.FontName(kit.ToString(vvv.Value.Interface()))
			FontChooserDialog(cb.Viewport, "Select a Font", "", nil, vv.This,
				func(recv, send ki.Ki, sig int64, data interface{}) {
					sv, _ := send.(*TableView)
					si := sv.SelectedIdx
					if si >= 0 {
						cbb := vvv.Widget.(*gi.Action)
						fi := gi.FontLibrary.FontInfo[si]
						vvv.SetValue(fi.Name)
						cbb.SetProp("font-family", fi.Name)
						cbb.SetFullReRender()
						cbb.SetText(fi.Name)
					}
				}, nil)
		}
	})
}
