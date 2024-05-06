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

	// [Buffer] for A
	BufferA *Buffer `json:"-" xml:"-"`

	// [Buffer] for B
	BufferB *Buffer `json:"-" xml:"-"`
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
			})
		}
	})
}

// MakeBuffers ensures that the [Buffer]s are made, if nil.
func (te *TwinEditors) MakeBuffers() {
	if te.BufferA != nil {
		return
	}
	te.BufferA = NewBuffer()
	te.BufferB = NewBuffer()
}

// SetFiles sets files for each text [Buffer].
func (te *TwinEditors) SetFiles(fileA, fileB string, LineNumbers bool) {
	te.MakeBuffers()
	te.BufferA.Filename = core.Filename(fileA)
	te.BufferA.Options.LineNumbers = LineNumbers
	te.BufferA.Stat() // update markup
	te.BufferB.Filename = core.Filename(fileB)
	te.BufferB.Options.LineNumbers = LineNumbers
	te.BufferB.Stat() // update markup
}

func (te *TwinEditors) ConfigTexts() {
	if te.HasChildren() {
		return
	}
	te.MakeBuffers()
	av := NewEditor(te, "text-a")
	bv := NewEditor(te, "text-b")
	av.SetBuffer(te.BufferA)
	bv.SetBuffer(te.BufferB)

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

// Editors returns the two text [Editor]s.
func (te *TwinEditors) Editors() (*Editor, *Editor) {
	ae := te.Child(0).(*Editor)
	be := te.Child(1).(*Editor)
	return ae, be
}
