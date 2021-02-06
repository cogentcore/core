// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"io/ioutil"
	"log"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv/textbuf"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/token"
	"github.com/goki/vci"
)

// DiffFiles shows the diffs between this file as the A file, and other file as B file,
// in a DiffViewDialog
func DiffFiles(afile, bfile string) (*DiffView, error) {
	ab, err := ioutil.ReadFile(afile)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	bb, err := ioutil.ReadFile(bfile)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	astr := strings.Split(strings.Replace(string(ab), "\r\n", "\n", -1), "\n") // windows safe
	bstr := strings.Split(strings.Replace(string(bb), "\r\n", "\n", -1), "\n")
	dlg := DiffViewDialog(nil, astr, bstr, afile, bfile, "", "", DlgOpts{Title: "Diff File View:"})
	return dlg, nil
}

// DiffViewDialogFromRevs opens a dialog for displaying diff between file
// at two different revisions from given repository
// if empty, defaults to: A = current HEAD, B = current WC file.
// -1, -2 etc also work as universal ways of specifying prior revisions.
func DiffViewDialogFromRevs(avp *gi.Viewport2D, repo vci.Repo, file string, fbuf *TextBuf, rev_a, rev_b string) (*DiffView, error) {
	var astr, bstr []string
	if rev_b == "" { // default to current file
		if fbuf != nil {
			bstr = fbuf.Strings(false)
		} else {
			fb, err := textbuf.FileBytes(file)
			if err != nil {
				return nil, err
			}
			bstr = textbuf.BytesToLineStrings(fb, false) // don't add new lines
		}
	} else {
		fb, err := repo.FileContents(file, rev_b)
		if err != nil {
			return nil, err
		}
		bstr = textbuf.BytesToLineStrings(fb, false) // don't add new lines
	}
	fb, err := repo.FileContents(file, rev_a)
	if err != nil {
		return nil, err
	}
	astr = textbuf.BytesToLineStrings(fb, false) // don't add new lines
	if rev_a == "" {
		rev_a = "HEAD"
	}
	return DiffViewDialog(nil, astr, bstr, file, file, rev_a, rev_b, DlgOpts{Title: "DiffVcs: " + DirAndFile(file)}), nil
}

// DiffViewDialog opens a dialog for displaying diff between two files as line-strings
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

///////////////////////////////////////////////////////////////////
// DiffView

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
	UndoA  textbuf.Diffs `json:"-" xml:"-" desc:"undo diffs records aligned diffs with edits applied"`
	UndoB  textbuf.Diffs `json:"-" xml:"-" desc:"undo diffs records aligned diffs with edits applied"`
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
	for {
		di++
		if di >= nd {
			return false
		}
		df = dv.AlignD[di]
		if df.Tag != 'e' {
			break
		}
	}
	tva.SetCursorShow(lex.Pos{Ln: df.I1})
	tva.ScrollCursorToVertCenter()
	tvb.SetCursorShow(lex.Pos{Ln: df.I1})
	tvb.ScrollCursorToVertCenter()
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
	for {
		di--
		if di < 0 {
			return false
		}
		df = dv.AlignD[di]
		if df.Tag != 'e' {
			break
		}
	}
	tva.SetCursorShow(lex.Pos{Ln: df.I1})
	tva.ScrollCursorToVertCenter()
	tvb.SetCursorShow(lex.Pos{Ln: df.I1})
	tvb.ScrollCursorToVertCenter()
	return true
}

// ResetDiffs resets all active diff state -- after saving
func (dv *DiffView) ResetDiffs() {
	dv.BufA.LineColors = nil
	dv.BufB.LineColors = nil
	dv.AlignD = nil
	dv.EditA = nil
	dv.UndoA = nil
	dv.EditB = nil
	dv.UndoB = nil
}

// RemoveAlignsA removes extra blank text lines added to align with B
func (dv *DiffView) RemoveAlignsA() {
	nd := len(dv.EditA)
	for i := nd - 1; i >= 0; i-- {
		df := dv.EditA[i]
		switch df.Tag {
		case 'r':
			if df.J2 > df.I2 {
				spos := lex.Pos{Ln: df.I2, Ch: 0}
				epos := lex.Pos{Ln: df.J2, Ch: 0}
				dv.BufA.DeleteText(spos, epos, true)
			}
		case 'i':
			spos := lex.Pos{Ln: df.J1, Ch: 0}
			epos := lex.Pos{Ln: df.J2, Ch: 0}
			dv.BufA.DeleteText(spos, epos, true)
		}
	}
}

// SaveFileA saves the current state of file A to given filename
func (dv *DiffView) SaveFileA(fname gi.FileName) {
	dv.RemoveAlignsA()
	dv.RemoveAlignsB()
	dv.ResetDiffs()
	dv.BufA.SaveAs(fname)
	dv.UpdateToolBar()
}

// RemoveAlignsB removes extra blank text lines added to align with A
func (dv *DiffView) RemoveAlignsB() {
	nd := len(dv.EditB)
	for i := nd - 1; i >= 0; i-- {
		df := dv.EditB[i]
		switch df.Tag {
		case 'r':
			if df.I2 > df.J2 {
				spos := lex.Pos{Ln: df.J2, Ch: 0}
				epos := lex.Pos{Ln: df.I2, Ch: 0}
				dv.BufB.DeleteText(spos, epos, true)
			}
		case 'd':
			spos := lex.Pos{Ln: df.I1, Ch: 0}
			epos := lex.Pos{Ln: df.I2, Ch: 0}
			dv.BufB.DeleteText(spos, epos, true)
		}
	}
}

// SaveFileB saves the current state of file B to given filename
func (dv *DiffView) SaveFileB(fname gi.FileName) {
	dv.RemoveAlignsA()
	dv.RemoveAlignsB()
	dv.ResetDiffs()
	dv.BufB.SaveAs(fname)
	dv.UpdateToolBar()
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

// ApplyDiff applies change from the other buffer to the buffer for given file
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
		spos := lex.Pos{Ln: df.I1, Ch: 0}
		epos := lex.Pos{Ln: df.I2, Ch: 0}
		src := dv.BufB.Region(spos, epos)
		dv.BufA.DeleteText(spos, epos, true)
		dv.BufA.InsertText(spos, src.ToBytes(), true) // we always just copy, is blank for delete..
		ei, _ := dv.EditA.DiffForLine(df.J1)
		if ei >= 0 {
			dv.UndoA = append(dv.UndoA, dv.EditA[ei])
			if df.Tag == 'd' {
				dv.EditA[ei].Tag = 'i' // switch to insert which means that it will delete at end
			} else {
				dv.EditA = append(dv.EditA[:ei], dv.EditA[ei+1:]...)
			}
		}
	} else {
		dv.BufB.Undos.Off = false
		spos := lex.Pos{Ln: df.J1, Ch: 0}
		epos := lex.Pos{Ln: df.J2, Ch: 0}
		src := dv.BufA.Region(spos, epos)
		dv.BufB.DeleteText(spos, epos, true)
		dv.BufB.InsertText(spos, src.ToBytes(), true)
		ei, _ := dv.EditB.DiffForLine(df.I1)
		if ei >= 0 {
			dv.UndoB = append(dv.UndoB, dv.EditB[ei])
			if df.Tag == 'i' {
				dv.EditB[ei].Tag = 'd' // switch to delete..
			} else {
				dv.EditB = append(dv.EditB[:ei], dv.EditB[ei+1:]...)
			}
		}
	}
	dv.UpdateToolBar()
	return true
}

// UndoDiff undoes last applied change, if any -- just does Undo in buffer and
// updates the list of edits applied.
func (dv *DiffView) UndoDiff(ab int) {
	tva, tvb := dv.TextViews()
	tv := tva
	if ab == 1 {
		tv = tvb
	}
	if ab == 0 {
		dv.BufA.Undos.Off = false
		nd := len(dv.UndoA)
		if nd == 0 {
			return
		}
		df := dv.UndoA[nd-1]
		dv.UndoA = dv.UndoA[:nd-1]
		if df.Tag == 'd' {
			ei, _ := dv.EditA.DiffForLine(df.J1) // existing record
			if ei >= 0 {
				dv.EditA[ei].Tag = 'd' // restore
			}
		} else {
			ei, _ := dv.EditA.DiffForLine(df.J1 - 1) // place to insert
			oi, od := dv.AlignD.DiffForLine(df.J1)
			if oi >= 0 {
				df.Tag = od.Tag // restore
			}
			if ei < 0 {
				dv.EditA = append(textbuf.Diffs{df}, dv.EditA...)
			} else {
				tmp := append(dv.EditA[:ei+1], df)
				tmp = append(tmp, dv.EditA[ei+1:]...)
				dv.EditA = tmp
			}
		}
	} else {
		dv.BufB.Undos.Off = false
		nd := len(dv.UndoB)
		if nd == 0 {
			return
		}
		df := dv.UndoB[nd-1]
		dv.UndoB = dv.UndoB[:nd-1]
		if df.Tag == 'i' {
			ei, _ := dv.EditB.DiffForLine(df.J1) // existing record
			if ei >= 0 {
				dv.EditB[ei].Tag = 'i' // restore
			}
		} else {
			ei, _ := dv.EditB.DiffForLine(df.J1 - 1) // place to insert
			oi, od := dv.AlignD.DiffForLine(df.J1)
			if oi >= 0 {
				df.Tag = od.Tag // restore
			}
			if ei < 0 {
				dv.EditB = append(textbuf.Diffs{df}, dv.EditB...)
			} else {
				tmp := append(dv.EditB[:ei+1], df)
				tmp = append(tmp, dv.EditB[ei+1:]...)
				dv.EditB = tmp
			}
		}
	}
	tv.Undo()
	dv.UpdateToolBar()
}

func (dv *DiffView) Config() {
	dv.Lay = gi.LayoutVert
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_Layout, "diff-lay")
	mods, updt := dv.ConfigChildren(config)
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

func (dv *DiffView) HasDiffsUpdate(act *gi.Action) {
	act.SetActiveStateUpdt(len(dv.AlignD) > 1) // always has at least 1
}

func (dv *DiffView) ConfigToolBar() {
	tb := dv.ToolBar()
	tb.SetStretchMaxWidth()
	txta := "A: " + DirAndFile(dv.FileA)
	if dv.RevA != "" {
		txta += ": " + dv.RevA
	}
	gi.AddNewLabel(tb, "label-a", txta)
	tb.AddAction(gi.ActOpts{Label: "Next", Icon: "wedge-down", Tooltip: "move down to next diff region", UpdateFunc: dv.HasDiffsUpdate},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.NextDiff(0)
		})
	tb.AddAction(gi.ActOpts{Label: "Prev", Icon: "wedge-up", Tooltip: "move up to previous diff region", UpdateFunc: dv.HasDiffsUpdate},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.PrevDiff(0)
		})
	tb.AddAction(gi.ActOpts{Label: "A <- B", Icon: "copy", Tooltip: "for current diff region, apply change from corresponding version in B, and move to next diff", UpdateFunc: dv.HasDiffsUpdate},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.ApplyDiff(0, -1)
			dvv.NextDiff(0)
		})
	tb.AddAction(gi.ActOpts{Label: "Undo", Icon: "undo", Tooltip: "undo last diff apply action (A <- B)", UpdateFunc: dv.FileModifiedUpdateA},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.UndoDiff(0)
		})
	tb.AddAction(gi.ActOpts{Label: "Save", Icon: "file-save", Tooltip: "save edited version of file -- prompts for filename -- this will convert file back to its original form (removing side-by-side alignment) and end the diff editing function", UpdateFunc: dv.FileModifiedUpdateA},
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
	tb.AddAction(gi.ActOpts{Label: "Next", Icon: "wedge-down", Tooltip: "move down to next diff region", UpdateFunc: dv.HasDiffsUpdate},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.NextDiff(1)
		})
	tb.AddAction(gi.ActOpts{Label: "Prev", Icon: "wedge-up", Tooltip: "move up to previous diff region", UpdateFunc: dv.HasDiffsUpdate},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.PrevDiff(1)
		})
	tb.AddAction(gi.ActOpts{Label: "A -> B", Icon: "copy", Tooltip: "for current diff region, apply change from corresponding version in A, and move to next diff", UpdateFunc: dv.HasDiffsUpdate},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.ApplyDiff(1, -1)
			dvv.NextDiff(1)
		})
	tb.AddAction(gi.ActOpts{Label: "Undo", Icon: "undo", Tooltip: "undo last diff apply action (A -> B)", UpdateFunc: dv.FileModifiedUpdateB},
		dv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DiffView).(*DiffView)
			dvv.UndoDiff(1)
		})
	tb.AddAction(gi.ActOpts{Label: "Save", Icon: "file-save", Tooltip: "save edited version of file -- prompts for filename -- this will convert file back to its original form (removing side-by-side alignment) and end the diff editing function", UpdateFunc: dv.FileModifiedUpdateB},
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

func (dv *DiffView) UpdateToolBar() {
	tb := dv.ToolBar()
	tb.UpdateActions()
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
	mods, updt := lay.ConfigChildren(config)
	al, bl := dv.TextViewLays()
	if !mods {
		updt = lay.UpdateStart()
	} else {
		al.SetStretchMax()
		al.SetMinPrefWidth(units.NewCh(80))
		al.SetMinPrefHeight(units.NewEm(40))
		bl.SetStretchMax()
		bl.SetMinPrefWidth(units.NewCh(80))
		bl.SetMinPrefHeight(units.NewEm(40))

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
