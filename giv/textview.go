// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/textview"
	"goki.dev/gti"
	"goki.dev/laser"
)

// TextValue presents a [textview.View] for longer text
type TextValue struct {
	ValueBase
}

func (vv *TextValue) WidgetType() *gti.Type {
	vv.WidgetTyp = textview.ViewType
	return vv.WidgetTyp
}

func (vv *TextValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*textview.View)
	npv := laser.NonPtrValue(vv.Value)
	sb.Buf.SetText([]byte(npv.String()))
}

func (vv *TextValue) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)

	tb := textview.NewBuf()
	tb.Stat()

	tv := widg.(*textview.View)
	tv.SetBuf(tb)

	vv.UpdateWidget()
}
