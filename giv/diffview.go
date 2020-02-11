// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv/textbuf"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

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
		dv.BufA = NewTextBuf()
		dv.BufB = NewTextBuf()
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
		al.SetMinPrefWidth(units.NewCh(20))
		al.SetMinPrefHeight(units.NewCh(10))
		bl.SetStretchMax()
		bl.SetMinPrefWidth(units.NewCh(20))
		bl.SetMinPrefHeight(units.NewCh(10))

		av := AddNewTextView(al, "text-a")
		bv := AddNewTextView(bl, "text-b")
		av.SetInactive()
		bv.SetInactive()
		av.SetBuf(dv.BufA)
		bv.SetBuf(dv.BufB)
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
	for _, df := range dv.Diffs {
		switch df.Tag {
		case 'r':
			di := df.I2 - df.I1
			dj := df.J2 - df.J1
			mx := ints.MaxInt(di, dj)
			for i := 0; i < mx; i++ {
				dv.BufA.SetLineColor(absln+i, chg)
				dv.BufB.SetLineColor(absln+i, chg)
				if i < di {
					ab = append(ab, []byte(astr[df.I1+i]))
				} else {
					ab = append(ab, []byte(""))
				}
				if i < dj {
					bb = append(bb, []byte(bstr[df.J1+i]))
				} else {
					bb = append(bb, []byte(""))
				}
			}
			absln += mx
		case 'd':
			di := df.I2 - df.I1
			for i := 0; i < di; i++ {
				dv.BufA.SetLineColor(absln+i, del)
				dv.BufB.SetLineColor(absln+i, ins)
				ab = append(ab, []byte(astr[df.I1+i]))
				bb = append(bb, []byte(""))
			}
			absln += di
		case 'i':
			dj := df.J2 - df.J1
			for i := 0; i < dj; i++ {
				dv.BufA.SetLineColor(absln+i, ins)
				dv.BufB.SetLineColor(absln+i, del)
				bb = append(bb, []byte(bstr[df.J1+i]))
				ab = append(ab, []byte(""))
			}
			absln += dj
		case 'e':
			di := df.I2 - df.I1
			dj := df.J2 - df.J1
			mx := ints.MaxInt(di, dj) // must be the same!
			for i := 0; i < mx; i++ {
				if i < di {
					ab = append(ab, []byte(astr[df.I1+i]))
				} else {
					ab = append(ab, []byte(""))
				}
				if i < dj {
					bb = append(bb, []byte(bstr[df.J1+i]))
				} else {
					bb = append(bb, []byte(""))
				}
			}
			absln += mx
		}
	}
	dv.BufA.SetTextLines(ab, false) // don't copy
	dv.BufB.SetTextLines(bb, false) // don't copy
	av.UpdateEnd(aupdt)
	bv.UpdateEnd(bupdt)
}

// DiffViewProps are style properties for DiffView
var DiffViewProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"max-width":     -1,
	"max-height":    -1,
}

// DiffViewDialog opens a dialog for displaying diff between two strings
func DiffViewDialog(avp *gi.Viewport2D, astr, bstr []string, afile, bfile string, opts DlgOpts) *DiffView {
	dlg := gi.NewStdDialog(opts.ToGiOpts(), opts.Ok, opts.Cancel)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	dv := frame.InsertNewChild(KiT_DiffView, prIdx+1, "diff-view").(*DiffView)
	dv.SetProp("width", units.NewEm(5))
	dv.SetProp("height", units.NewEm(5))
	dv.SetStretchMax()
	dv.FileA = afile
	dv.FileB = bfile
	dv.DiffStrings(astr, bstr)

	// bbox, _ := dlg.ButtonBox(frame)
	// if bbox == nil {
	// 	bbox = dlg.AddButtonBox(frame)
	// }
	// cpb := gi.AddNewButton(bbox, "copy-to-clip")
	// cpb.SetText("Copy To Clipboard")
	// cpb.SetIcon("copy")
	// cpb.ButtonSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	// 	if sig == int64(gi.ButtonClicked) {
	// 		ddlg := recv.Embed(gi.KiT_Dialog).(*gi.Dialog)
	// 		oswin.TheApp.ClipBoard(ddlg.Win.OSWin).Write(mimedata.NewTextBytes(text))
	// 	}
	// })

	dlg.UpdateEndNoSig(true) // going to be shown
	dlg.Open(0, 0, avp, nil)
	return dv
}
