// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv/textbuf"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/token"
)

// todo: double-click on diff region copies over corresponding values from
// other side!  Not clear how to connect this back to original Buf though,
// in context of gide.

// DiffView presents two side-by-side TextView windows showing the differences
// between two files (represented as lines of strings).
type DiffView struct {
	gi.Frame
	FileA  string        `desc:"first file name being compared"`
	FileB  string        `desc:"second file name being compared"`
	RevA   string        `desc:"revision for first file, if relevant"`
	RevB   string        `desc:"revision for second file, if relevant"`
	Diffs  textbuf.Diffs `json:"-" xml:"-" desc:"the diff records"`
	BufA   *TextBuf      `json:"-" xml:"-" desc:"textbuf for A"`
	BufB   *TextBuf      `json:"-" xml:"-" desc:"textbuf for B"`
	AlignD textbuf.Diffs `json:"-" xml:"-" desc:"aligned diffs records diff for aligned lines"`
	EditA  textbuf.Diffs `json:"-" xml:"-" desc:"edit diffs records aligned diffs with edits applied"`
	EditB  textbuf.Diffs `json:"-" xml:"-" desc:"edit diffs records aligned diffs with edits applied"`
}

var KiT_DiffView = kit.Types.AddType(&DiffView{}, DiffViewProps)

// AddNewDiffView adds a new diffview to given parent node, with given name.
func AddNewDiffView(parent ki.Ki, name string) *DiffView {
	return parent.AddNewChild(KiT_DiffView, name).(*DiffView)
}

// NextDiff moves to next diff region
func (dv *DiffView) NextDiff(ab int) bool {
	tva, tvb := dv.TextViews()
	tv := tva
	if ab == 1 {
		tv = tvb
	}
	nd := len(dv.AlignD)
	curLn := tv.CursorPos.Ln
	di, df := dv.AlignD.DiffForLine(curLn)
	if di < 0 {
		return false
	}
	di++
	if di >= nd {
		return false
	}
	df = dv.AlignD[di]
	tva.SetCursorShow(textbuf.Pos{Ln: df.I1})
	tvb.SetCursorShow(textbuf.Pos{Ln: df.I1})
	return true
}

// PrevDiff moves to previous diff region
func (dv *DiffView) PrevDiff(ab int) bool {
	tva, tvb := dv.TextViews()
	tv := tva
	if ab == 1 {
		tv = tvb
	}
	curLn := tv.CursorPos.Ln
	di, df := dv.AlignD.DiffForLine(curLn)
	if di < 0 {
		return false
	}
	di--
	if di < 0 {
		return false
	}
	df = dv.AlignD[di]
	tva.SetCursorShow(textbuf.Pos{Ln: df.I1})
	tvb.SetCursorShow(textbuf.Pos{Ln: df.I1})
	return true
}

func (dv *DiffView) Undo(ab int) {
	tva, tvb := dv.TextViews()
	tv := tva
	if ab == 1 {
		tv = tvb
	}
	tv.Buf.Undo()
}

// RemoveAlignsA removes extra blank text lines added to align with B
func (dv *DiffView) RemoveAlignsA() {
	nd := len(dv.EditA)
	for i := nd - 1; i >= 0; i-- {
		df := dv.EditA[i]
		switch df.Tag {
		case 'r':
			if df.J2 > df.I2 {
				spos := textbuf.Pos{Ln: df.I2, Ch: 0}
				epos := textbuf.Pos{Ln: df.J2, Ch: 0}
				dv.BufA.DeleteText(spos, epos, true)
			}
		case 'i':
			spos := textbuf.Pos{Ln: df.J1, Ch: 0}
			epos := textbuf.Pos{Ln: df.J2, Ch: 0}
			dv.BufA.DeleteText(spos, epos, true)
		}
	}
	dv.BufA.LineColors = nil
	dv.BufB.LineColors = nil
}

// SaveFileA saves the current state of file A to given filename
func (dv *DiffView) SaveFileA(fname gi.FileName) {
	dv.RemoveAlignsA()
	dv.BufA.SaveAs(fname)
}

// RemoveAlignsB removes extra blank text lines added to align with A
func (dv *DiffView) RemoveAlignsB() {
	nd := len(dv.EditB)
	for i := nd - 1; i >= 0; i-- {
		df := dv.EditB[i]
		switch df.Tag {
		case 'r':
			if df.I2 > df.J2 {
				spos := textbuf.Pos{Ln: df.J2, Ch: 0}
				epos := textbuf.Pos{Ln: df.I2, Ch: 0}
				dv.BufB.DeleteText(spos, epos, true)
			}
		case 'd':
			spos := textbuf.Pos{Ln: df.I1, Ch: 0}
			epos := textbuf.Pos{Ln: df.I2, Ch: 0}
			dv.BufB.DeleteText(spos, epos, true)
		}
	}
	dv.BufA.LineColors = nil
	dv.BufB.LineColors = nil
}

// SaveFileB saves the current state of file B to given filename
func (dv *DiffView) SaveFileB(fname gi.FileName) {
	dv.RemoveAlignsB()
	dv.BufB.SaveAs(fname)
}

// DiffStrings computes differences between two lines-of-strings and displays in
// DiffView.
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
	nd := len(dv.Diffs)
	dv.AlignD = make(textbuf.Diffs, nd)
	dv.EditA = make(textbuf.Diffs, nd)
	dv.EditB = make(textbuf.Diffs, nd)
	var ab, bb [][]byte
	absln := 0
	bspc := []byte(" ")
	for i, df := range dv.Diffs {
		switch df.Tag {
		case 'r':
			di := df.I2 - df.I1
			dj := df.J2 - df.J1
			mx := ints.MaxInt(di, dj)
			ad := df
			ad.I1 = absln
			ad.I2 = absln + di
			ad.J1 = absln
			ad.J2 = absln + dj
			dv.AlignD[i] = ad
			dv.EditA[i] = ad
			dv.EditB[i] = ad
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
			ad := df
			ad.I1 = absln
			ad.I2 = absln + di
			ad.J1 = absln
			ad.J2 = absln + di
			dv.AlignD[i] = ad
			dv.EditA[i] = ad
			dv.EditB[i] = ad
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
			ad := df
			ad.I1 = absln
			ad.I2 = absln + dj
			ad.J1 = absln
			ad.J2 = absln + dj
			dv.AlignD[i] = ad
			dv.EditA[i] = ad
			dv.EditB[i] = ad
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
			ad := df
			ad.I1 = absln
			ad.I2 = absln + di
			ad.J1 = absln
			ad.J2 = absln + di
			dv.AlignD[i] = ad
			dv.EditA[i] = ad
			dv.EditB[i] = ad
			for i := 0; i < di; i++ {
				ab = append(ab, []byte(astr[df.I1+i]))
				bb = append(bb, []byte(bstr[df.J1+i]))
			}
			absln += di
		}
	}
	dv.BufA.SetTextLines(ab, false) // don't copy
	dv.BufB.SetTextLines(bb, false) // don't copy
	dv.TagWordDiffs()
	dv.BufA.ReMarkup()
	dv.BufB.ReMarkup()
	av.UpdateEnd(aupdt)
	bv.UpdateEnd(bupdt)
}

// TagWordDiffs goes through replace diffs and tags differences at the
// word level between the two regions.
func (dv *DiffView) TagWordDiffs() {
	for _, df := range dv.AlignD {
		if df.Tag != 'r' {
			continue
		}
		di := df.I2 - df.I1
		dj := df.J2 - df.J1
		mx := ints.MaxInt(di, dj)
		stln := df.I1
		for i := 0; i < mx; i++ {
			ln := stln + i
			ra := dv.BufA.Lines[ln]
			rb := dv.BufB.Lines[ln]
			lna := lex.RuneFields(ra)
			lnb := lex.RuneFields(rb)
			fla := lna.RuneStrings(ra)
			flb := lnb.RuneStrings(rb)
			nab := ints.MaxInt(len(fla), len(flb))
			ldif := textbuf.DiffLines(fla, flb)
			ndif := len(ldif)
			if nab > 25 && ndif > nab/2 { // more than half of big diff -- skip
				continue
			}
			for _, ld := range ldif {
				switch ld.Tag {
				case 'r':
					sla := lna[ld.I1]
					ela := lna[ld.I2-1]
					dv.BufA.AddTag(ln, sla.St, ela.Ed, token.TextStyleError)
					slb := lnb[ld.J1]
					elb := lnb[ld.J2-1]
					dv.BufB.AddTag(ln, slb.St, elb.Ed, token.TextStyleError)
				case 'd':
					sla := lna[ld.I1]
					ela := lna[ld.I2-1]
					dv.BufA.AddTag(ln, sla.St, ela.Ed, token.TextStyleDeleted)
				case 'i':
					slb := lnb[ld.J1]
					elb := lnb[ld.J2-1]
					dv.BufB.AddTag(ln, slb.St, elb.Ed, token.TextStyleDeleted)
				}
			}
		}
	}
}

// ApplyDiff copies the text from the other buffer to the buffer for given file
// name, from diff that includes given line.
func (dv *DiffView) ApplyDiff(ab int, line int) bool {
	tva, tvb := dv.TextViews()
	tv := tva
	if ab == 1 {
		tv = tvb
	}
	if line < 0 {
		line = tv.CursorPos.Ln
	}
	di, df := dv.AlignD.DiffForLine(line)
	if di < 0 || df.Tag == 'e' {
		return false
	}
	if ab == 0 {
		dv.BufA.Undos.Off = false
		// srcLen := len(dv.BufB.Lines[df.J2])
		spos := textbuf.Pos{Ln: df.J1, Ch: 0}
		epos := textbuf.Pos{Ln: df.J2, Ch: 0}
		src := dv.BufB.Region(spos, epos)
		dv.BufA.DeleteText(spos, epos, true)
		dv.BufA.InsertText(spos, src.ToBytes(), true)
		for ln := df.I1; ln < df.I2; ln++ {
			dv.BufA.DeleteLineColor(ln)
		}
		ei, _ := dv.EditA.DiffForLine(df.J1)
		if ei >= 0 {
			dv.EditA = append(dv.EditA[:ei], dv.EditA[ei+1:]...)
		}
	} else {
		dv.BufB.Undos.Off = false
		spos := textbuf.Pos{Ln: df.I1, Ch: 0}
		epos := textbuf.Pos{Ln: df.I2, Ch: 0}
		src := dv.BufA.Region(spos, epos)
		dv.BufB.DeleteText(spos, epos, true)
		dv.BufB.InsertText(spos, src.ToBytes(), true)
		for ln := df.I1; ln < df.I2; ln++ {
			dv.BufB.DeleteLineColor(ln)
		}
		ei, _ := dv.EditB.DiffForLine(df.I1)
		if ei >= 0 {
			dv.EditB = append(dv.EditB[:ei], dv.EditB[ei+1:]...)
		}
	}
	return true
}

func (dv *DiffView) Config() {
	dv.Lay = gi.LayoutVert
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_Layout, "diff-lay")
	mods, updt := dv.ConfigChildren(config, ki.UniqueNames)
	if !mods {
		updt = dv.UpdateStart()
		dv.SetTextNames()
	} else {
		dv.ConfigToolBar()
		dv.ConfigTexts()
	}
	dv.SetFullReRender()
	dv.UpdateEnd(updt)
}

func (dv *DiffView) FileModifiedUpdateA(act *gi.Action) {
	act.SetActiveStateUpdt(dv.BufA.IsChanged())
}

func (dv *DiffView) FileModifiedUpdateB(act *gi.Action) {
	act.SetActiveStateUpdt(dv.BufB.IsChanged())
}

func (dv *DiffView) ConfigToolBar() {
	tb := dv.ToolBar()
	tb.SetStretchMaxWidth()
	txta := "A: " + DirAndFile(dv.FileA)
	if dv.RevA != "" {
		txta += ": " + dv.RevA
	}
	gi.AddNewLabel(tb, "label-a", txta)
	tb.AddAction(gi.ActOpts{Label: "Next", Icon: "wedge-down", Tooltip: "move down to next diff region"},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.NextDiff(0)
		})
	tb.AddAction(gi.ActOpts{Label: "Prev", Icon: "wedge-up", Tooltip: "move up to previous diff region"},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.PrevDiff(0)
		})
	tb.AddAction(gi.ActOpts{Label: "A <- B", Icon: "copy", Tooltip: "for current diff region, copy from corresponding version in B"},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.ApplyDiff(0, -1)
		})
	tb.AddAction(gi.ActOpts{Label: "Undo", Icon: "undo", Tooltip: "undo any edits made by applying diffs through double-clicking on difference regions", UpdateFunc: dv.FileModifiedUpdateA},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.Undo(0)
		})
	tb.AddAction(gi.ActOpts{Label: "Save", Icon: "file-save", Tooltip: "save edited version of file -- prompts for filename", UpdateFunc: dv.FileModifiedUpdateA},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			CallMethod(dvv, "SaveFileA", dv.Viewport)
		})
	gi.AddNewStretch(tb, "str")

	txtb := "B: " + DirAndFile(dv.FileB)
	if dv.RevB != "" {
		txtb += ": " + dv.RevB
	}
	gi.AddNewLabel(tb, "label-b", txtb)
	tb.AddAction(gi.ActOpts{Label: "Next", Icon: "wedge-down", Tooltip: "move down to next diff region"},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.NextDiff(1)
		})
	tb.AddAction(gi.ActOpts{Label: "Prev", Icon: "wedge-up", Tooltip: "move up to previous diff region"},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.PrevDiff(1)
		})
	tb.AddAction(gi.ActOpts{Label: "A -> B", Icon: "copy", Tooltip: "for current diff region, copy from corresponding version in A"},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.ApplyDiff(1, -1)
		})
	tb.AddAction(gi.ActOpts{Label: "Undo", Icon: "undo", Tooltip: "undo any edits made by applying diffs through double-clicking on difference regions", UpdateFunc: dv.FileModifiedUpdateB},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.Undo(1)
		})
	tb.AddAction(gi.ActOpts{Label: "Save", Icon: "file-save", Tooltip: "save edited version of file -- prompts for filename", UpdateFunc: dv.FileModifiedUpdateB},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			CallMethod(dvv, "SaveFileB", dv.Viewport)
		})
}

func (dv *DiffView) SetTextNames() {
	tb := dv.ToolBar()
	la := tb.ChildByName("label-a", 0).(*gi.Label)
	txta := "A: " + DirAndFile(dv.FileA)
	if dv.RevA != "" {
		txta += ": " + dv.RevA
	}
	la.SetText(txta)
	lb := tb.ChildByName("label-b", 4).(*gi.Label)
	txtb := "B: " + DirAndFile(dv.FileB)
	if dv.RevB != "" {
		txtb += ": " + dv.RevB
	}
	lb.SetText(txtb)
}

func (dv *DiffView) ToolBar() *gi.ToolBar {
	tb := dv.ChildByName("toolbar", 1).(*gi.ToolBar)
	return tb
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

func (dv *DiffView) TextViews() (*DiffTextView, *DiffTextView) {
	a, b := dv.TextViewLays()
	av := a.Child(0).(*DiffTextView)
	bv := b.Child(0).(*DiffTextView)
	return av, bv
}

func (dv *DiffView) ConfigTexts() {
	lay := dv.DiffLay()
	if dv.BufA == nil {
		dv.BufA = &TextBuf{}
		dv.BufA.InitName(dv.BufA, "diff-buf-a")
		dv.BufB = &TextBuf{}
		dv.BufB.InitName(dv.BufB, "diff-buf-b")
	}
	dv.BufA.Filename = gi.FileName(dv.FileA)
	dv.BufA.Opts.LineNos = true
	dv.BufA.Stat() // update markup
	dv.BufB.Filename = gi.FileName(dv.FileB)
	dv.BufB.Opts.LineNos = true
	dv.BufB.Stat() // update markup
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

		av := AddNewDiffTextView(al, "text-a")
		bv := AddNewDiffTextView(bl, "text-b")
		av.SetProp("font-family", gi.Prefs.MonoFont)
		bv.SetProp("font-family", gi.Prefs.MonoFont)
		// av.SetInactive()
		// bv.SetInactive()
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

// DiffViewProps are style properties for DiffView
var DiffViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"max-width":        -1,
	"max-height":       -1,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"CallMethods": ki.PropSlice{
		{"SaveFileA", ki.Props{
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "FileA",
				}},
			},
		}},
		{"SaveFileB", ki.Props{
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "FileB",
				}},
			},
		}},
	},
}

// DiffViewDialog opens a dialog for displaying diff between two strings
func DiffViewDialog(avp *gi.Viewport2D, astr, bstr []string, afile, bfile, arev, brev string, opts DlgOpts) *DiffView {
	dlg := gi.NewStdDialog(opts.ToGiOpts(), opts.Ok, opts.Cancel)

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	dv := frame.InsertNewChild(KiT_DiffView, prIdx+1, "diff-view").(*DiffView)
	// dv.SetProp("width", units.NewEm(20))
	// dv.SetProp("height", units.NewEm(10))
	dv.SetStretchMax()
	dv.FileA = afile
	dv.FileB = bfile
	dv.RevA = arev
	dv.RevB = brev
	dv.DiffStrings(astr, bstr)

	dlg.UpdateEndNoSig(true) // going to be shown
	dlg.Open(0, 0, avp, nil)
	return dv
}

////////////////////////////////////////////////////////////////////////////////
//   DiffTextView

// DiffTextView supports double-click based application of edits from one
// buffer to the other.
type DiffTextView struct {
	TextView
}

var KiT_DiffTextView = kit.Types.AddType(&DiffTextView{}, TextViewProps)

// AddNewDiffTextView adds a new DiffTextView to given parent node, with given name.
func AddNewDiffTextView(parent ki.Ki, name string) *DiffTextView {
	return parent.AddNewChild(KiT_DiffTextView, name).(*DiffTextView)
}

func (tv *DiffTextView) DiffView() *DiffView {
	dvi := tv.ParentByType(KiT_DiffView, ki.NoEmbeds)
	if dvi == nil {
		return nil
	}
	return dvi.(*DiffView)
}

// MouseEvent handles the mouse.Event to process double-click
func (tv *DiffTextView) MouseEvent(me *mouse.Event) {
	if me.Button != mouse.Left || me.Action != mouse.DoubleClick {
		tv.TextView.MouseEvent(me)
		return
	}
	pt := tv.PointToRelPos(me.Pos())
	if pt.X >= 0 && pt.X < int(tv.LineNoOff) {
		newPos := tv.PixelToCursor(pt)
		ln := newPos.Ln
		dv := tv.DiffView()
		if dv != nil && tv.Buf != nil {
			if tv.Nm == "text-a" {
				dv.ApplyDiff(0, ln)
			} else {
				dv.ApplyDiff(1, ln)
			}
		}
		tv.RenderLines(ln, ln)
		me.SetProcessed()
		return
	}
	tv.TextView.MouseEvent(me)
}

// TextViewEvents sets connections between mouse and key events and actions
func (tv *DiffTextView) TextViewEvents() {
	tv.HoverTooltipEvent()
	tv.MouseMoveEvent()
	tv.MouseDragEvent()
	tv.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_DiffTextView).(*DiffTextView)
		me := d.(*mouse.Event)
		txf.MouseEvent(me) // gets our new one
	})
	tv.MouseFocusEvent()
	tv.ConnectEvent(oswin.KeyChordEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_TextView).(*TextView)
		kt := d.(*key.ChordEvent)
		txf.KeyInput(kt)
	})
}

// ConnectEvents2D indirectly sets connections between mouse and key events and actions
func (tv *DiffTextView) ConnectEvents2D() {
	tv.TextViewEvents()
}
