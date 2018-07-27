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

// todo: manual
// ValueView returns the ValueView representation for the icon name --
// presents a chooser
// func (inm IconName) ValueView() ValueView {
// 	vv := IconValueView{}
// 	vv.Init(&vv)
// 	return &vv
// }

////////////////////////////////////////////////////////////////////////////////////////
//  IconValueView

// IconValueView presents a StructViewInline for a struct plus a IconView button..
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

	cb.SetIcon(txt)
}

func (vv *IconValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg

	cb := vv.Widget.(*gi.Action)
	vv.UpdateWidget()

	cb.ActionSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_IconValueView).(*IconValueView)
		if !vvv.IsInactive() {
			// cbb := vvv.Widget.(*Action)
			// eval := cbb.CurVal.(string)
			// if vvv.SetIcon(eval) {
			// vvv.UpdateWidget()
			// }
		}
	})
}
