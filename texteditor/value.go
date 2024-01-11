// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"

	"goki.dev/events"
	"goki.dev/gi"
	"goki.dev/giv"
	"goki.dev/grr"
	"goki.dev/gti"
	"goki.dev/laser"
)

// Value is a [texteditor.Editor] [giv.Value] for editing longer text
type Value struct {
	giv.ValueBase
}

func (vv *Value) WidgetType() *gti.Type {
	vv.WidgetTyp = EditorType
	return vv.WidgetTyp
}

func (vv *Value) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	te := vv.Widget.(*Editor)
	npv := laser.NonPtrValue(vv.Value)
	te.Buf.SetText([]byte(npv.String()))
}

func (vv *Value) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)

	tb := NewBuf()
	grr.Log(tb.Stat())
	tb.OnChange(func(e events.Event) {
		// fmt.Println(tb.Txt)
		vv.SetValue(string(tb.Txt))
		fmt.Println(laser.OnePtrUnderlyingValue(vv.Value).Interface())
	})

	te := w.(*Editor)
	te.SetBuf(tb)

	vv.UpdateWidget()
}
