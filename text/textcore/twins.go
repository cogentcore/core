// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/tree"
)

// TwinEditors presents two side-by-side [Editor]s in [core.Splits]
// that scroll in sync with each other.
type TwinEditors struct {
	core.Splits

	// [Buffer] for A
	BufferA *lines.Lines `json:"-" xml:"-"`

	// [Buffer] for B
	BufferB *lines.Lines `json:"-" xml:"-"`

	inInputEvent bool
}

func (te *TwinEditors) Init() {
	te.Splits.Init()
	te.BufferA = lines.NewLines()
	te.BufferB = lines.NewLines()

	f := func(name string, buf *lines.Lines) {
		tree.AddChildAt(te, name, func(w *Editor) {
			w.SetLines(buf)
			w.Styler(func(s *styles.Style) {
				s.Min.X.Ch(80)
				s.Min.Y.Em(40)
			})
			w.On(events.Scroll, func(e events.Event) {
				te.syncEditors(events.Scroll, e, name)
			})
			w.On(events.Input, func(e events.Event) {
				te.syncEditors(events.Input, e, name)
			})
		})
	}
	f("text-a", te.BufferA)
	f("text-b", te.BufferB)
}

// SetFiles sets the files for each [Buffer].
func (te *TwinEditors) SetFiles(fileA, fileB string) {
	te.BufferA.SetFilename(fileA)
	te.BufferA.Stat() // update markup
	te.BufferB.SetFilename(fileB)
	te.BufferB.Stat() // update markup
}

// syncEditors synchronizes the [Editor] scrolling and cursor positions
func (te *TwinEditors) syncEditors(typ events.Types, e events.Event, name string) {
	tva, tvb := te.Editors()
	me, other := tva, tvb
	if name == "text-b" {
		me, other = tvb, tva
	}
	switch typ {
	case events.Scroll:
		other.Geom.Scroll.Y = me.Geom.Scroll.Y
		other.ScrollUpdateFromGeom(math32.Y)
	case events.Input:
		if te.inInputEvent {
			return
		}
		te.inInputEvent = true
		other.SetCursorShow(me.CursorPos)
		te.inInputEvent = false
	}
}

// Editors returns the two text [Editor]s.
func (te *TwinEditors) Editors() (*Editor, *Editor) {
	ae := te.Child(0).(*Editor)
	be := te.Child(1).(*Editor)
	return ae, be
}
