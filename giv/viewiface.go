// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/gi/v2/texteditor/histyle"
	"goki.dev/ki/v2"
)

// giv.ViewIFace is THE implementation of the gi.ViewIFace interface
type ViewIFace struct {
}

func (vi *ViewIFace) CtxtMenuView(val any, readOnly bool, sc *gi.Scene, m *gi.Scene) bool {
	// TODO(kai/menu): add back CtxtMenuView here
	// return CtxtMenuView(val, readOnly, sc, menu)
	return false
}

func (vi *ViewIFace) CallFunc(ctx gi.Widget, fun any) {
	CallFunc(ctx, fun)
}

func (vi *ViewIFace) Inspector(obj ki.Ki) {
	InspectorDialog(obj)
}

func (vi *ViewIFace) PrefsView(prefs *gi.Preferences) {
	PrefsView(prefs)
}

func (vi *ViewIFace) KeyMapsView(maps *keyfun.Maps) {
	KeyMapsView(maps)
}

func (vi *ViewIFace) PrefsDetView(prefs *gi.PrefsDetailed) {
	PrefsDetView(prefs)
}

func (vi *ViewIFace) HiStylesView(std bool) {
	if std {
		HiStylesView(&histyle.StdStyles)
	} else {
		HiStylesView(&histyle.CustomStyles)
	}
}

func (vi *ViewIFace) HiStyleInit() {
	histyle.Init()
}

func (vi *ViewIFace) SetHiStyleDefault(hsty gi.HiStyleName) {
	histyle.StyleDefault = hsty
}

func (vi *ViewIFace) PrefsDetDefaults(pf *gi.PrefsDetailed) {
	// pf.TextEditorClipHistMax = TextEditorClipHistMax
	// pf.TextBufMaxScopeLines = TextBufMaxScopeLines
	// pf.TextBufDiffRevertLines = TextBufDiffRevertLines
	// pf.TextBufDiffRevertDiffs = TextBufDiffRevertDiffs
	// pf.TextBufMarkupDelayMSec = TextBufMarkupDelayMSec
	pf.MapInlineLen = MapInlineLen
	pf.StructInlineLen = StructInlineLen
	pf.SliceInlineLen = SliceInlineLen
}

func (vi *ViewIFace) PrefsDetApply(pf *gi.PrefsDetailed) {
	// TextEditorClipHistMax = pf.TextEditorClipHistMax
	// TextBufMaxScopeLines = pf.TextBufMaxScopeLines
	// TextBufDiffRevertLines = pf.TextBufDiffRevertLines
	// TextBufDiffRevertDiffs = pf.TextBufDiffRevertDiffs
	// TextBufMarkupDelayMSec = pf.TextBufMarkupDelayMSec
	MapInlineLen = pf.MapInlineLen
	StructInlineLen = pf.StructInlineLen
	SliceInlineLen = pf.SliceInlineLen
}

func (vi *ViewIFace) PrefsDbgView(prefs *gi.PrefsDebug) {
	PrefsDbgView(prefs)
}
