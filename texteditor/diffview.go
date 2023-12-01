// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/glop/dirs"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/token"
	"goki.dev/vci/v2"
)

// DiffFiles shows the diffs between this file as the A file, and other file as B file,
// in a DiffViewDialog
func DiffFiles(ctx gi.Widget, afile, bfile string) (*DiffView, error) {
	ab, err := os.ReadFile(afile)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	bb, err := os.ReadFile(bfile)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	astr := strings.Split(strings.Replace(string(ab), "\r\n", "\n", -1), "\n") // windows safe
	bstr := strings.Split(strings.Replace(string(bb), "\r\n", "\n", -1), "\n")
	dlg := DiffViewDialog(ctx, "Diff File View", astr, bstr, afile, bfile, "", "")
	return dlg, nil
}

// DiffViewDialogFromRevs opens a dialog for displaying diff between file
// at two different revisions from given repository
// if empty, defaults to: A = current HEAD, B = current WC file.
// -1, -2 etc also work as universal ways of specifying prior revisions.
func DiffViewDialogFromRevs(ctx gi.Widget, repo vci.Repo, file string, fbuf *Buf, rev_a, rev_b string) (*DiffView, error) {
	var astr, bstr []string
	if rev_b == "" { // default to current file
		if fbuf != nil {
			bstr = fbuf.Strings(false)
		} else {
			fb, err := textbuf.FileBytes(file)
			if err != nil {
				gi.ErrorDialog(ctx, err)
				return nil, err
			}
			bstr = textbuf.BytesToLineStrings(fb, false) // don't add new lines
		}
	} else {
		fb, err := repo.FileContents(file, rev_b)
		if err != nil {
			gi.ErrorDialog(ctx, err)
			return nil, err
		}
		bstr = textbuf.BytesToLineStrings(fb, false) // don't add new lines
	}
	fb, err := repo.FileContents(file, rev_a)
	if err != nil {
		gi.ErrorDialog(ctx, err)
		return nil, err
	}
	astr = textbuf.BytesToLineStrings(fb, false) // don't add new lines
	if rev_a == "" {
		rev_a = "HEAD"
	}
	return DiffViewDialog(ctx, "DiffVcs: "+dirs.DirAndFile(file), astr, bstr, file, file, rev_a, rev_b), nil
}

// DiffViewDialog opens a dialog for displaying diff between two files as line-strings
func DiffViewDialog(ctx gi.Widget, title string, astr, bstr []string, afile, bfile, arev, brev string) *DiffView {
	d := gi.NewBody("diffview")
	d.SetTitle(title)

	dv := NewDiffView(d, "diff-view")
	dv.FileA = afile
	dv.FileB = bfile
	dv.RevA = arev
	dv.RevB = brev
	dv.ConfigDiffView()
	dv.DiffStrings(astr, bstr)
	d.AddTopAppBar(dv.DiffViewTopAppBar)
	// todo: any buttons?
	d.NewWindow().SetContext(ctx).SetNewWindow(true).Run()
	return dv
}

///////////////////////////////////////////////////////////////////
// DiffView

// DiffView presents two side-by-side TextEditor windows showing the differences
// between two files (represented as lines of strings).
type DiffView struct {
	gi.Frame

	// first file name being compared
	FileA string

	// second file name being compared
	FileB string

	// revision for first file, if relevant
	RevA string

	// revision for second file, if relevant
	RevB string

	// textbuf for A showing the aligned edit view
	BufA *Buf `json:"-" xml:"-" set:"-"`

	// textbuf for B showing the aligned edit view
	BufB *Buf `json:"-" xml:"-" set:"-"`

	// aligned diffs records diff for aligned lines
	AlignD textbuf.Diffs `json:"-" xml:"-" set:"-"`

	// Diffs applied
	Diffs textbuf.DiffSelected
}

func (dv *DiffView) OnInit() {
	dv.DiffViewStyles()
}

func (dv *DiffView) DiffViewStyles() {
	dv.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	dv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(dv) {
		case "text-a", "text-b":
			w.Style(func(s *styles.Style) {
				s.Font.Family = string(gi.Prefs.MonoFont)
				s.Min.X.Ch(80)
				s.Min.Y.Em(40)
			})
		}
	})
}

// NextDiff moves to next diff region
func (dv *DiffView) NextDiff(ab int) bool {
	tva, tvb := dv.TextEditors()
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
	tva, tvb := dv.TextEditors()
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

// SaveAs saves A or B edits into given file.
// It checks for an existing file, prompts to overwrite or not.
func (dv *DiffView) SaveAs(ab bool, filename gi.FileName) {
	if !grr.Log1(dirs.FileExists(string(filename))) {
		dv.SaveFile(ab, filename)
	} else {
		d := gi.NewBody().AddTitle("File Exists, Overwrite?").
			AddText(fmt.Sprintf("File already exists, overwrite?  File: %v", filename))
		d.AddBottomBar(func(pw gi.Widget) {
			d.AddCancel(pw)
			d.AddOk(pw).OnClick(func(e events.Event) {
				dv.SaveFile(ab, filename)
			})
		})
		d.NewDialog(dv).Run()
	}
}

// SaveFile writes A or B edits to file, with no prompting, etc
func (dv *DiffView) SaveFile(ab bool, filename gi.FileName) error {
	var txt string
	if ab {
		txt = strings.Join(dv.Diffs.B.Edit, "\n")
	} else {
		txt = strings.Join(dv.Diffs.A.Edit, "\n")
	}
	err := os.WriteFile(string(filename), []byte(txt), 0644)
	if err != nil {
		gi.ErrorSnackbar(dv, err)
		slog.Error(err.Error())
	}
	return err
}

// SaveFileA saves the current state of file A to given filename
func (dv *DiffView) SaveFileA(fname gi.FileName) { //gti:add
	dv.SaveAs(false, fname)
	// dv.UpdateToolbar()
}

// SaveFileB saves the current state of file B to given filename
func (dv *DiffView) SaveFileB(fname gi.FileName) { //gti:add
	dv.SaveAs(true, fname)
	// dv.UpdateToolbar()
}

// DiffStrings computes differences between two lines-of-strings and displays in
// DiffView.
func (dv *DiffView) DiffStrings(astr, bstr []string) {
	av, bv := dv.TextEditors()
	aupdt := av.UpdateStart()
	bupdt := bv.UpdateStart()

	dv.Diffs.SetStringLines(astr, bstr)

	dv.BufA.LineColors = nil
	dv.BufB.LineColors = nil
	del := colors.Scheme.Error.Base
	ins := colors.Scheme.Success.Base
	chg := colors.Scheme.Primary.Base

	nd := len(dv.Diffs.Diffs)
	dv.AlignD = make(textbuf.Diffs, nd)
	var ab, bb [][]byte
	absln := 0
	bspc := []byte(" ")
	for i, df := range dv.Diffs.Diffs {
		switch df.Tag {
		case 'r':
			di := df.I2 - df.I1
			dj := df.J2 - df.J1
			mx := max(di, dj)
			ad := df
			ad.I1 = absln
			ad.I2 = absln + di
			ad.J1 = absln
			ad.J2 = absln + dj
			dv.AlignD[i] = ad
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
	av.UpdateEndRender(aupdt)
	bv.UpdateEndRender(bupdt)
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
		mx := max(di, dj)
		stln := df.I1
		for i := 0; i < mx; i++ {
			ln := stln + i
			ra := dv.BufA.Lines[ln]
			rb := dv.BufB.Lines[ln]
			lna := lex.RuneFields(ra)
			lnb := lex.RuneFields(rb)
			fla := lna.RuneStrings(ra)
			flb := lnb.RuneStrings(rb)
			nab := max(len(fla), len(flb))
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
	tva, tvb := dv.TextEditors()
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
		dv.Diffs.BtoA(di)
	} else {
		dv.BufB.Undos.Off = false
		spos := lex.Pos{Ln: df.J1, Ch: 0}
		epos := lex.Pos{Ln: df.J2, Ch: 0}
		src := dv.BufA.Region(spos, epos)
		dv.BufB.DeleteText(spos, epos, true)
		dv.BufB.InsertText(spos, src.ToBytes(), true)
		dv.Diffs.AtoB(di)
	}
	// dv.UpdateToolbar()
	return true
}

// UndoDiff undoes last applied change, if any.
func (dv *DiffView) UndoDiff(ab int) error {
	tva, tvb := dv.TextEditors()
	if ab == 1 {
		if !dv.Diffs.B.Undo() {
			err := errors.New("No more edits to undo")
			gi.ErrorSnackbar(dv, err)
			return err
		}
		tvb.Undo()
	} else {
		if !dv.Diffs.A.Undo() {
			err := errors.New("No more edits to undo")
			gi.ErrorSnackbar(dv, err)
			return err
		}
		tva.Undo()
	}
	return nil
}

func (dv *DiffView) ConfigWidget() {
	dv.ConfigDiffView()
}

func (dv *DiffView) ConfigDiffView() {
	if dv.HasChildren() {
		return
	}
	dv.BufA = NewBuf().SetFilename(dv.FileA)
	dv.BufB = NewBuf().SetFilename(dv.FileB)
	dv.BufA.Opts.LineNos = true
	dv.BufA.Stat() // update markup
	dv.BufB.Opts.LineNos = true
	dv.BufB.Stat() // update markup
	av := NewDiffTextEditor(dv, "text-a").SetBuf(dv.BufA)
	av.SetReadOnly(true)
	bv := NewDiffTextEditor(dv, "text-b").SetBuf(dv.BufB)
	bv.SetReadOnly(true)

	av.On(events.Scroll, func(e events.Event) {
		// bv.ScrollDelta(e)
		bv.Geom.Scroll.Y = av.Geom.Scroll.Y
		bv.SetNeedsRender(true)
	})
	bv.On(events.Scroll, func(e events.Event) {
		// av.ScrollDelta(e)
		av.Geom.Scroll.Y = bv.Geom.Scroll.Y
		av.SetNeedsRender(true)
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

// DiffViewTopAppBar configures the top app bar with diff view actions
func (dv *DiffView) DiffViewTopAppBar(tb *gi.TopAppBar) {
	txta := "A: " + dirs.DirAndFile(dv.FileA)
	if dv.RevA != "" {
		txta += ": " + dv.RevA
	}
	gi.NewLabel(tb, "label-a").SetText(txta)
	gi.NewButton(tb).SetText("Next").SetIcon(icons.KeyboardArrowDown).
		SetTooltip("move down to next diff region").
		OnClick(func(e events.Event) {
			dv.NextDiff(0)
		}).
		Style(func(s *styles.Style) {
			s.State.SetFlag(len(dv.AlignD) <= 1, states.Disabled)
		})
	gi.NewButton(tb).SetText("Prev").SetIcon(icons.KeyboardArrowUp).
		SetTooltip("move up to previous diff region").
		OnClick(func(e events.Event) {
			dv.PrevDiff(0)
		}).
		Style(func(s *styles.Style) {
			s.State.SetFlag(len(dv.AlignD) <= 1, states.Disabled)
		})
	gi.NewButton(tb).SetText("A &lt;- B").SetIcon(icons.ContentCopy).
		SetTooltip("for current diff region, apply change from corresponding version in B, and move to next diff").
		OnClick(func(e events.Event) {
			dv.ApplyDiff(0, -1)
			dv.NextDiff(0)
		}).
		Style(func(s *styles.Style) {
			s.State.SetFlag(len(dv.AlignD) <= 1, states.Disabled)
		})
	gi.NewButton(tb).SetText("Undo").SetIcon(icons.Undo).
		SetTooltip("undo last diff apply action (A &lt;- B)").
		OnClick(func(e events.Event) {
			dv.UndoDiff(0)
		}).
		Style(func(s *styles.Style) {
			s.State.SetFlag(!dv.BufA.IsChanged(), states.Disabled)
		})
	gi.NewButton(tb).SetText("Save").SetIcon(icons.Save).
		SetTooltip("save edited version of file with the given -- prompts for filename").
		OnClick(func(e events.Event) {
			// fb := giv.NewSoloFuncButton(ctx, dv.SaveFileA)
			// fb.Args[0].SetValue(dv.FileA)
			// fb.CallFunc()
			gi.TheViewIFace.CallFunc(dv, dv.SaveFileA)
		}).
		Style(func(s *styles.Style) {
			s.State.SetFlag(!dv.BufA.IsChanged(), states.Disabled)
		})

	gi.NewSeparator(tb)

	txtb := "B: " + dirs.DirAndFile(dv.FileB)
	if dv.RevB != "" {
		txtb += ": " + dv.RevB
	}
	gi.NewLabel(tb, "label-b").SetText(txtb)
	gi.NewButton(tb).SetText("Next").SetIcon(icons.KeyboardArrowDown).
		SetTooltip("move down to next diff region").
		OnClick(func(e events.Event) {
			dv.NextDiff(1)
		}).
		Style(func(s *styles.Style) {
			s.State.SetFlag(len(dv.AlignD) <= 1, states.Disabled)
		})
	gi.NewButton(tb).SetText("Prev").SetIcon(icons.KeyboardArrowUp).
		SetTooltip("move up to previous diff region").
		OnClick(func(e events.Event) {
			dv.PrevDiff(1)
		}).
		Style(func(s *styles.Style) {
			s.State.SetFlag(len(dv.AlignD) <= 1, states.Disabled)
		})
	gi.NewButton(tb).SetText("A -&gt; B").SetIcon(icons.ContentCopy).
		SetTooltip("for current diff region, apply change from corresponding version in A, and move to next diff").
		OnClick(func(e events.Event) {
			dv.ApplyDiff(1, -1)
			dv.NextDiff(1)
		}).
		Style(func(s *styles.Style) {
			s.State.SetFlag(len(dv.AlignD) <= 1, states.Disabled)
		})
	gi.NewButton(tb).SetText("Undo").SetIcon(icons.Undo).
		SetTooltip("undo last diff apply action (A -&gt; B)").
		OnClick(func(e events.Event) {
			dv.UndoDiff(1)
		}).
		Style(func(s *styles.Style) {
			s.State.SetFlag(!dv.BufB.IsChanged(), states.Disabled)
		})
	gi.NewButton(tb).SetText("Save").SetIcon(icons.Save).
		SetTooltip("save edited version of file -- prompts for filename -- this will convert file back to its original form (removing side-by-side alignment) and end the diff editing function").
		OnClick(func(e events.Event) {
			// fb := giv.NewSoloFuncButton(ctx, dv.SaveFileB)
			// fb.Args[0].SetValue(dv.FileB)
			// fb.CallFunc()
			gi.TheViewIFace.CallFunc(dv, dv.SaveFileB)
		}).
		Style(func(s *styles.Style) {
			s.State.SetFlag(!dv.BufB.IsChanged(), states.Disabled)
		})
}

func (dv *DiffView) TextEditors() (*DiffTextEditor, *DiffTextEditor) {
	av := dv.Child(0).(*DiffTextEditor)
	bv := dv.Child(1).(*DiffTextEditor)
	return av, bv
}

////////////////////////////////////////////////////////////////////////////////
//   DiffTextEditor

// DiffTextEditor supports double-click based application of edits from one
// buffer to the other.
type DiffTextEditor struct {
	Editor
}

func (tv *DiffTextEditor) OnInit() {
	tv.Editor.OnInit()
	tv.HandleDiffDoubleClick()
}

func (tv *DiffTextEditor) DiffView() *DiffView {
	dvi := tv.ParentByType(DiffViewType, ki.NoEmbeds)
	if dvi == nil {
		return nil
	}
	return dvi.(*DiffView)
}

func (tv *DiffTextEditor) HandleDiffDoubleClick() {
	tv.On(events.DoubleClick, func(e events.Event) {
		pt := tv.PointToRelPos(e.LocalPos())
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
			e.SetHandled()
			return
		}
	})
}
