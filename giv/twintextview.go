// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

///////////////////////////////////////////////////////////////////
// TwinTextViews

// TwinTextViews presents two side-by-side TextView windows in SplitView
// that scroll in sync with each other.
type TwinTextViews struct {
	gi.SplitView

	// textbuf for A
	BufA *TextBuf `json:"-" xml:"-" desc:"textbuf for A"`

	// textbuf for B
	BufB *TextBuf `json:"-" xml:"-" desc:"textbuf for B"`
}

func (tv *TwinTextViews) OnInit() {
	tv.Dim = mat32.X
	tv.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		s.BackgroundColor.SetSolid(colors.Scheme.Background)
		s.Color = colors.Scheme.OnBackground
		s.SetStretchMax()
	})
}

func (tv *TwinTextViews) OnChildAdded(child ki.Ki) {
	if w := gi.AsWidget(child); w != nil {
		switch w.Name() {
		case "text-a-lay", "text-b-lay":
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.SetStretchMax()
				s.SetMinPrefWidth(units.Ch(80))
				s.SetMinPrefHeight(units.Em(40))
			})
		case "text-a", "text-b":
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.Font.Family = string(gi.Prefs.MonoFont)
			})
		}
	}
}

// MakeBufs ensures that the TextBufs are made, if nil
func (tv *TwinTextViews) MakeBufs() {
	if tv.BufA != nil {
		return
	}
	tv.BufA = &TextBuf{}
	tv.BufA.InitName(tv.BufA, "buf-a")
	tv.BufB = &TextBuf{}
	tv.BufB.InitName(tv.BufB, "buf-b")
}

// SetFiles sets files for each text buf
func (tv *TwinTextViews) SetFiles(fileA, fileB string, lineNos bool) {
	tv.MakeBufs()
	tv.BufA.Filename = gi.FileName(fileA)
	tv.BufA.Opts.LineNos = lineNos
	tv.BufA.Stat() // update markup
	tv.BufB.Filename = gi.FileName(fileB)
	tv.BufB.Opts.LineNos = lineNos
	tv.BufB.Stat() // update markup
}

func (tv *TwinTextViews) ConfigTexts() {
	tv.MakeBufs()
	config := ki.TypeAndNameList{}
	config.Add(gi.LayoutType, "text-a-lay")
	config.Add(gi.LayoutType, "text-b-lay")
	mods, updt := tv.ConfigChildren(config)
	al, bl := tv.TextViewLays()
	if !mods {
		updt = tv.UpdateStart()
	} else {
		av := NewTextView(al, "text-a")
		bv := NewTextView(bl, "text-b")
		av.SetBuf(tv.BufA)
		bv.SetBuf(tv.BufB)

		// sync scrolling
		al.ScrollSig.Connect(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			dm := mat32.Dims(sig)
			if dm == mat32.Y {
				bl.ScrollToPos(dm, data.(float32))
			}
		})
		bl.ScrollSig.Connect(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			dm := mat32.Dims(sig)
			if dm == mat32.Y {
				al.ScrollToPos(dm, data.(float32))
			}
		})
	}
	tv.UpdateEnd(updt)
}

// TextViewLays returns the two layouts that control the two textviews
func (tv *TwinTextViews) TextViewLays() (*gi.Layout, *gi.Layout) {
	a := tv.Child(0).(*gi.Layout)
	b := tv.Child(1).(*gi.Layout)
	return a, b
}

// TextViews returns the two textviews
func (tv *TwinTextViews) TextViews() (*TextView, *TextView) {
	a, b := tv.TextViewLays()
	av := a.Child(0).(*TextView)
	bv := b.Child(0).(*TextView)
	return av, bv
}

// TwinTextViewsProps are style properties for TwinTextViews
var TwinTextViewsProps = ki.Props{
	ki.EnumTypeFlag: gi.TypeNodeFlags,
}
