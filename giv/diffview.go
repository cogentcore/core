// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv/textbuf"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// todo: double-click on diff region copies over corresponding values from
// other side!  Not clear how to connect this back to original Buf though,
// in context of gide.

// DiffView presents two side-by-side TextView windows showing the differences
// between two files (represented as lines of strings).
type DiffView struct {
	gi.Frame
	FileA string        `desc:"first file name being compared"`
	FileB string        `desc:"second file name being compared"`
	Diffs textbuf.Diffs `json:"-" xml:"-" desc:"the diff records"`
	BufA  *TextBuf      `json:"-" xml:"-" desc:"textbuf for A"`
	BufB  *TextBuf      `json:"-" xml:"-" desc:"textbuf for B"`
}

var KiT_DiffView = kit.Types.AddType(&DiffView{}, DiffViewProps)

// AddNewDiffView adds a new diffview to given parent node, with given name.
func AddNewDiffView(parent ki.Ki, name string) *DiffView {
	return parent.AddNewChild(KiT_DiffView, name).(*DiffView)
}

func (dv *DiffView) Config() {
	dv.Lay = gi.LayoutVert
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "label")
	config.Add(gi.KiT_Layout, "diff-lay")
	mods, updt := dv.ConfigChildren(config, ki.UniqueNames)
	if !mods {
		updt = dv.UpdateStart()
	}
	dv.SetLabel()
	dv.ConfigTexts()
	dv.SetFullReRender()
	dv.UpdateEnd(updt)
}

func (dv *DiffView) SetLabel() {
	label := dv.ChildByName("label", 0).(*gi.Label)
	label.SetProp("text-align", gi.AlignCenter)
	lbl := "Diff: A = " + dv.FileA + "  B = " + dv.FileB
	label.SetText(lbl)
}

func (dv *DiffView) DiffLay() *gi.Layout {
	lay := dv.ChildByName("diff-lay", 1).(*gi.Layout)
	return lay
}

func (dv *DiffView) TextViewLays() (*gi.Layout, *gi.Layout) {
	lay := dv.DiffLay()
	a := lay.Child(0).(*gi.Layout)
	b := lay.Child(1).(*gi.Layout)
	return a, b
}

func (dv *DiffView) TextViews() (*TextView, *TextView) {
	a, b := dv.TextViewLays()
	av := a.Child(0).(*TextView)
	bv := b.Child(0).(*TextView)
	return av, bv
}

func (dv *DiffView) ConfigTexts() {
	lay := dv.DiffLay()
	if dv.BufA == nil {
		dv.BufA = &TextBuf{}
		dv.BufA.InitName(dv.BufA, "diff-buf-a")
		dv.BufA.Filename = gi.FileName(dv.FileA)
		dv.BufA.Opts.LineNos = true
		dv.BufA.Stat() // update markup
		dv.BufB = &TextBuf{}
		dv.BufB.InitName(dv.BufB, "diff-buf-b")
		dv.BufB.Filename = gi.FileName(dv.FileB)
		dv.BufB.Opts.LineNos = true
		dv.BufB.Stat() // update markup
	}
	lay.Lay = gi.LayoutHoriz
	lay.SetStretchMax()
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Layout, "text-a-lay")
	config.Add(gi.KiT_Layout, "text-b-lay")
	mods, updt := lay.ConfigChildren(config, ki.UniqueNames)
	al, bl := dv.TextViewLays()
	if !mods {
		updt = lay.UpdateStart()
	} else {
		al.SetStretchMax()
		al.SetMinPrefWidth(units.NewEm(10))
		al.SetMinPrefHeight(units.NewEm(10))
		bl.SetStretchMax()
		bl.SetMinPrefWidth(units.NewEm(10))
		bl.SetMinPrefHeight(units.NewEm(10))

		av := AddNewTextView(al, "text-a")
		bv := AddNewTextView(bl, "text-b")
		av.SetProp("font-family", gi.Prefs.MonoFont)
		bv.SetProp("font-family", gi.Prefs.MonoFont)
		av.SetInactive()
		bv.SetInactive()
		av.SetBuf(dv.BufA)
		bv.SetBuf(dv.BufB)

		// sync scrolling
		al.ScrollSig.Connect(dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			bl.ScrollToPos(mat32.Dims(sig), data.(float32))
		})
		bl.ScrollSig.Connect(dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			al.ScrollToPos(mat32.Dims(sig), data.(float32))
		})
	}
	lay.UpdateEnd(updt)
}

func (dv *DiffView) IsConfiged() bool {
	if dv.NumChildren() > 0 && dv.BufA != nil {
		return true
	}
	return false
}

func (dv *DiffView) DiffStrings(astr, bstr []string) {
	if !dv.IsConfiged() {
		dv.Config()
	}
	av, bv := dv.TextViews()
	aupdt := av.UpdateStart()
	bupdt := bv.UpdateStart()
	dv.BufA.LineColors = nil
	dv.BufB.LineColors = nil
	del := "red"
	ins := "green"
	chg := "blue"
	dv.Diffs = textbuf.DiffLines(astr, bstr)
	var ab, bb [][]byte
	absln := 0
	bspc := []byte(" ")
	for _, df := range dv.Diffs {
		switch df.Tag {
		case 'r':
			di := df.I2 - df.I1
			dj := df.J2 - df.J1
			mx := ints.MaxInt(di, dj)
			for i := 0; i < mx; i++ {
				dv.BufA.SetLineColor(absln+i, chg)
				dv.BufB.SetLineColor(absln+i, chg)
				blen := 0
				alen := 0
				if i < di {
					aln := []byte(astr[df.I1+i])
					alen = len(aln)
					ab = append(ab, aln)
				}
				if i < dj {
					bln := []byte(bstr[df.J1+i])
					blen = len(bln)
					bb = append(bb, bln)
				} else {
					bb = append(bb, bytes.Repeat(bspc, alen))
				}
				if i >= di {
					ab = append(ab, bytes.Repeat(bspc, blen))
				}
			}
			absln += mx
		case 'd':
			di := df.I2 - df.I1
			for i := 0; i < di; i++ {
				dv.BufA.SetLineColor(absln+i, del)
				dv.BufB.SetLineColor(absln+i, ins)
				aln := []byte(astr[df.I1+i])
				alen := len(aln)
				ab = append(ab, aln)
				bb = append(bb, bytes.Repeat(bspc, alen))
			}
			absln += di
		case 'i':
			dj := df.J2 - df.J1
			for i := 0; i < dj; i++ {
				dv.BufA.SetLineColor(absln+i, ins)
				dv.BufB.SetLineColor(absln+i, del)
				bln := []byte(bstr[df.J1+i])
				blen := len(bln)
				bb = append(bb, bln)
				ab = append(ab, bytes.Repeat(bspc, blen))
			}
			absln += dj
		case 'e':
			di := df.I2 - df.I1
			for i := 0; i < di; i++ {
				ab = append(ab, []byte(astr[df.I1+i]))
				bb = append(bb, []byte(bstr[df.J1+i]))
			}
			absln += di
		}
	}
	dv.BufA.SetTextLines(ab, false) // don't copy
	dv.BufB.SetTextLines(bb, false) // don't copy
	dv.BufA.ReMarkup()
	dv.BufB.ReMarkup()
	av.UpdateEnd(aupdt)
	bv.UpdateEnd(bupdt)
}

// DiffViewProps are style properties for DiffView
var DiffViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"max-width":        -1,
	"max-height":       -1,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
}

// DiffViewDialog opens a dialog for displaying diff between two strings
func DiffViewDialog(avp *gi.Viewport2D, astr, bstr []string, afile, bfile string, opts DlgOpts) *DiffView {
	dlg := gi.NewStdDialog(opts.ToGiOpts(), opts.Ok, opts.Cancel)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	dv := frame.InsertNewChild(KiT_DiffView, prIdx+1, "diff-view").(*DiffView)
	// dv.SetProp("width", units.NewEm(20))
	// dv.SetProp("height", units.NewEm(10))
	dv.SetStretchMax()
	dv.FileA = afile
	dv.FileB = bfile
	dv.DiffStrings(astr, bstr)

	dlg.UpdateEndNoSig(true) // going to be shown
	dlg.Open(0, 0, avp, nil)
	return dv
}
