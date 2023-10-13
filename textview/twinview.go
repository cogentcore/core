// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textview

import (
	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// TwinViews presents two side-by-side View windows in Splits
// that scroll in sync with each other.
type TwinViews struct {
	gi.Splits

	// textbuf for A
	BufA *Buf `json:"-" xml:"-"`

	// textbuf for B
	BufB *Buf `json:"-" xml:"-"`
}

func (tv *TwinViews) OnInit() {
	tv.Dim = mat32.X
	tv.AddStyles(func(s *styles.Style) {
		s.BackgroundColor.SetSolid(colors.Scheme.Background)
		s.Color = colors.Scheme.OnBackground
		s.SetStretchMax()
	})
}

func (tv *TwinViews) OnChildAdded(child ki.Ki) {
	w, _ := gi.AsWidget(child)
	switch w.Name() {
	case "text-a-lay", "text-b-lay":
		w.AddStyles(func(s *styles.Style) {
			s.SetStretchMax()
			s.SetMinPrefWidth(units.Ch(80))
			s.SetMinPrefHeight(units.Em(40))
		})
	case "text-a", "text-b":
		w.AddStyles(func(s *styles.Style) {
			s.Font.Family = string(gi.Prefs.MonoFont)
		})
	}
}

// MakeBufs ensures that the Bufs are made, if nil
func (tv *TwinViews) MakeBufs() {
	if tv.BufA != nil {
		return
	}
	tv.BufA = NewBuf()
	tv.BufB = NewBuf()
}

// SetFiles sets files for each text buf
func (tv *TwinViews) SetFiles(fileA, fileB string, lineNos bool) {
	tv.MakeBufs()
	tv.BufA.Filename = gi.FileName(fileA)
	tv.BufA.Opts.LineNos = lineNos
	tv.BufA.Stat() // update markup
	tv.BufB.Filename = gi.FileName(fileB)
	tv.BufB.Opts.LineNos = lineNos
	tv.BufB.Stat() // update markup
}

func (tv *TwinViews) ConfigTexts() {
	tv.MakeBufs()
	config := ki.Config{}
	config.Add(gi.LayoutType, "text-a-lay")
	config.Add(gi.LayoutType, "text-b-lay")
	mods, updt := tv.ConfigChildren(config)
	al, bl := tv.ViewLays()
	if !mods {
		updt = tv.UpdateStart()
	} else {
		av := NewView(al, "text-a")
		bv := NewView(bl, "text-b")
		av.SetBuf(tv.BufA)
		bv.SetBuf(tv.BufB)

		// sync scrolling
		// al.ScrollSig.Connect(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
		// 	dm := mat32.Dims(sig)
		// 	if dm == mat32.Y {
		// 		bl.ScrollToPos(dm, data.(float32))
		// 	}
		// })
		// bl.ScrollSig.Connect(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
		// 	dm := mat32.Dims(sig)
		// 	if dm == mat32.Y {
		// 		al.ScrollToPos(dm, data.(float32))
		// 	}
		// })
	}
	tv.UpdateEnd(updt)
}

// ViewLays returns the two layouts that control the two textviews
func (tv *TwinViews) ViewLays() (*gi.Layout, *gi.Layout) {
	a := tv.Child(0).(*gi.Layout)
	b := tv.Child(1).(*gi.Layout)
	return a, b
}

// Views returns the two textviews
func (tv *TwinViews) Views() (*View, *View) {
	a, b := tv.ViewLays()
	av := a.Child(0).(*View)
	bv := b.Child(0).(*View)
	return av, bv
}
