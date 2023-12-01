// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/mat32/v2"
)

// TwinEditors presents two side-by-side [Editor]s in [gi.Splits]
// that scroll in sync with each other.
type TwinEditors struct {
	gi.Splits

	// textbuf for A
	BufA *Buf `json:"-" xml:"-"`

	// textbuf for B
	BufB *Buf `json:"-" xml:"-"`
}

func (te *TwinEditors) OnInit() {
	te.Dim = mat32.X
	te.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	te.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(te) {
		case "text-a", "text-b":
			w.Style(func(s *styles.Style) {
				s.Grow.Set(1, 1)
				s.Min.X.Ch(80)
				s.Min.Y.Em(40)
				s.Font.Family = string(gi.Prefs.MonoFont)
			})
		}
	})
}

// MakeBufs ensures that the Bufs are made, if nil
func (te *TwinEditors) MakeBufs() {
	if te.BufA != nil {
		return
	}
	te.BufA = NewBuf()
	te.BufB = NewBuf()
}

// SetFiles sets files for each text buf
func (te *TwinEditors) SetFiles(fileA, fileB string, lineNos bool) {
	te.MakeBufs()
	te.BufA.Filename = gi.FileName(fileA)
	te.BufA.Opts.LineNos = lineNos
	te.BufA.Stat() // update markup
	te.BufB.Filename = gi.FileName(fileB)
	te.BufB.Opts.LineNos = lineNos
	te.BufB.Stat() // update markup
}

func (te *TwinEditors) ConfigTexts() {
	if te.HasChildren() {
		return
	}
	te.MakeBufs()
	av := NewEditor(te, "text-a")
	bv := NewEditor(te, "text-b")
	av.SetBuf(te.BufA)
	bv.SetBuf(te.BufB)

	av.On(events.Scroll, func(e events.Event) {
		// bv.ScrollDelta(e)
		bv.Geom.Scroll.Y = av.Geom.Scroll.Y
		bv.SetNeedsRender(true)
	})
	bv.On(events.Scroll, func(e events.Event) {
		// av.ScrollDelta(e)
		av.Geom.Scroll.Y = bv.Geom.Scroll.Y
		av.SetNeedsRender(true)
	})
	inInputEvent := false
	av.On(events.Input, func(e events.Event) {
		if inInputEvent {
			return
		}
		inInputEvent = true
		bv.SetCursorShow(av.CursorPos)
		inInputEvent = false
	})
	bv.On(events.Input, func(e events.Event) {
		if inInputEvent {
			return
		}
		inInputEvent = true
		av.SetCursorShow(bv.CursorPos)
		inInputEvent = false
	})
}

// Editors returns the two text Editors
func (te *TwinEditors) Editors() (*Editor, *Editor) {
	ae := te.Child(0).(*Editor)
	be := te.Child(1).(*Editor)
	return ae, be
}
