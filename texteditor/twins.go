// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/ki/v2"
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
	te.MakeBufs()
	config := ki.Config{}
	config.Add(EditorType, "text-a")
	config.Add(EditorType, "text-b")
	mods, updt := te.ConfigChildren(config)
	if !mods {
		updt = te.UpdateStart()
	} else {
		ae, be := te.Editors()
		ae.SetBuf(te.BufA)
		be.SetBuf(te.BufB)

		// sync scrolling
		// al.ScrollSig.Connect(te.This(), func(recv, send ki.Ki, sig int64, data any) {
		// 	dm := mat32.Dims(sig)
		// 	if dm == mat32.Y {
		// 		bl.ScrollToPos(dm, data.(float32))
		// 	}
		// })
		// bl.ScrollSig.Connect(te.This(), func(recv, send ki.Ki, sig int64, data any) {
		// 	dm := mat32.Dims(sig)
		// 	if dm == mat32.Y {
		// 		al.ScrollToPos(dm, data.(float32))
		// 	}
		// })
	}
	te.UpdateEnd(updt)
}

// Editors returns the two text Editors
func (te *TwinEditors) Editors() (*Editor, *Editor) {
	ae := te.Child(0).(*Editor)
	be := te.Child(1).(*Editor)
	return ae, be
}
