// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "image"

// CtxtMenuFunc is a function for creating a context menu for given node
type CtxtMenuFunc func(g Widget, m *Menu)

func (wb *WidgetBase) MakeContextMenu(m *Menu) {
	// derived types put native menu code here
	if wb.CtxtMenuFunc != nil {
		wb.CtxtMenuFunc(wb.This().(Widget), m)
	}
	mvp := wb.Sc
	TheViewIFace.CtxtMenuView(wb.This(), wb.IsDisabled(), mvp, m)
}

func (wb *WidgetBase) ContextMenuPos() (pos image.Point) {
	wb.BBoxMu.RLock()
	pos.X = (wb.WinBBox.Min.X + wb.WinBBox.Max.X) / 2
	pos.Y = (wb.WinBBox.Min.Y + wb.WinBBox.Max.Y) / 2
	wb.BBoxMu.RUnlock()
	return
}

func (wb *WidgetBase) ContextMenu() {
	var men Menu
	wi := wb.This().(Widget)
	wi.MakeContextMenu(&men)
	if len(men) == 0 {
		return
	}
	pos := wi.ContextMenuPos()
	mvp := wb.Sc
	PopupMenu(men, pos.X, pos.Y, mvp, wb.Nm+"-menu")
}
