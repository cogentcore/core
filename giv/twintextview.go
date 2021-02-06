// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

///////////////////////////////////////////////////////////////////
// TwinTextViews

// TwinTextViews presents two side-by-side TextView windows in SplitView
// that scroll in sync with each other.
type TwinTextViews struct {
	gi.SplitView
	BufA *TextBuf `json:"-" xml:"-" desc:"textbuf for A"`
	BufB *TextBuf `json:"-" xml:"-" desc:"textbuf for B"`
}

var KiT_TwinTextViews = kit.Types.AddType(&TwinTextViews{}, TwinTextViewsProps)

// AddNewTwinTextViews adds a new diffview to given parent node, with given name.
func AddNewTwinTextViews(parent ki.Ki, name string) *TwinTextViews {
	return parent.AddNewChild(KiT_TwinTextViews, name).(*TwinTextViews)
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
	tv.Dim = mat32.X
	tv.SetStretchMax()
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Layout, "text-a-lay")
	config.Add(gi.KiT_Layout, "text-b-lay")
	mods, updt := tv.ConfigChildren(config)
	al, bl := tv.TextViewLays()
	if !mods {
		updt = tv.UpdateStart()
	} else {
		al.SetStretchMax()
		al.SetMinPrefWidth(units.NewCh(80))
		al.SetMinPrefHeight(units.NewEm(40))
		bl.SetStretchMax()
		bl.SetMinPrefWidth(units.NewCh(80))
		bl.SetMinPrefHeight(units.NewEm(40))

		av := AddNewTextView(al, "text-a")
		bv := AddNewTextView(bl, "text-b")
		av.SetProp("font-family", gi.Prefs.MonoFont)
		bv.SetProp("font-family", gi.Prefs.MonoFont)
		av.SetBuf(tv.BufA)
		bv.SetBuf(tv.BufB)

		// sync scrolling
		al.ScrollSig.Connect(tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dm := mat32.Dims(sig)
			if dm == mat32.Y {
				bl.ScrollToPos(dm, data.(float32))
			}
		})
		bl.ScrollSig.Connect(tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
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
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"max-width":        -1,
	"max-height":       -1,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
}
