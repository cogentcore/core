// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gti"
	"goki.dev/laser"
)

// TextEditorValue presents a [texteditor.Editor] for editing longer text
type TextEditorValue struct {
	ValueBase
}

func (vv *TextEditorValue) WidgetType() *gti.Type {
	vv.WidgetTyp = texteditor.ViewType
	return vv.WidgetTyp
}

func (vv *TextEditorValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*texteditor.Editor)
	npv := laser.NonPtrValue(vv.Value)
	sb.Buf.SetText([]byte(npv.String()))
}

func (vv *TextEditorValue) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)

	tb := texteditor.NewBuf()
	tb.Stat()

	tv := widg.(*texteditor.Editor)
	tv.SetBuf(tb)

	vv.UpdateWidget()
}
