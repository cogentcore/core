// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
)

// TwinEditors presents two side-by-side [Editor]s in [core.Splits]
// that scroll in sync with each other.
type TwinEditors struct {
	core.Splits

	// textbuf for A
	BufA *Buffer `json:"-" xml:"-"`

	// textbuf for B
	BufB *Buffer `json:"-" xml:"-"`
}

func (te *TwinEditors) OnInit() {
	te.Splits.OnInit()
	te.SetStyles()
}

func (te *TwinEditors) SetStyles() {
	te.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	te.OnWidgetAdded(func(w core.Widget) {
		switch w.PathFrom(te) {
		case "text-a", "text-b":
			w.Style(func(s *styles.Style) {
				s.Grow.Set(1, 1)
				s.Min.X.Ch(80)
				s.Min.Y.Em(40)
				s.Font.Family = string(core.AppearanceSettings.MonoFont)
			})
		}
	})
}

// MakeBufs ensures that the Bufs are made, if nil
func (te *TwinEditors) MakeBufs() {
	if te.BufA != nil {
		return
	}
	te.BufA = NewBuffer()
	te.BufB = NewBuffer()
}

// SetFiles sets files for each text buf
func (te *TwinEditors) SetFiles(fileA, fileB string, lineNos bool) {
	te.MakeBufs()
	te.BufA.Filename = core.Filename(fileA)
	te.BufA.Opts.LineNos = lineNos
	te.BufA.Stat() // update markup
	te.BufB.Filename = core.Filename(fileB)
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
	av.SetBuffer(te.BufA)
	bv.SetBuffer(te.BufB)

	av.On(events.Scroll, func(e events.Event) {
		// bv.ScrollDelta(e)
		bv.Geom.Scroll.Y = av.Geom.Scroll.Y
		bv.NeedsRender()
	})
	bv.On(events.Scroll, func(e events.Event) {
		// av.ScrollDelta(e)
		av.Geom.Scroll.Y = bv.Geom.Scroll.Y
		av.NeedsRender()
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
