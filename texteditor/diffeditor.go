// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/stringsx"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/parse/token"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/texteditor/text"
	"cogentcore.org/core/tree"
)

// DiffFiles shows the diffs between this file as the A file, and other file as B file,
// in a DiffEditorDialog
func DiffFiles(ctx core.Widget, afile, bfile string) (*DiffEditor, error) {
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
	astr := stringsx.SplitLines(string(ab))
	bstr := stringsx.SplitLines(string(bb))
	dlg := DiffEditorDialog(ctx, "Diff File View", astr, bstr, afile, bfile, "", "")
	return dlg, nil
}

// DiffEditorDialogFromRevs opens a dialog for displaying diff between file
// at two different revisions from given repository
// if empty, defaults to: A = current HEAD, B = current WC file.
// -1, -2 etc also work as universal ways of specifying prior revisions.
func DiffEditorDialogFromRevs(ctx core.Widget, repo vcs.Repo, file string, fbuf *Buffer, rev_a, rev_b string) (*DiffEditor, error) {
	var astr, bstr []string
	if rev_b == "" { // default to current file
		if fbuf != nil {
			bstr = fbuf.Strings(false)
		} else {
			fb, err := text.FileBytes(file)
			if err != nil {
				core.ErrorDialog(ctx, err)
				return nil, err
			}
			bstr = text.BytesToLineStrings(fb, false) // don't add new lines
		}
	} else {
		fb, err := repo.FileContents(file, rev_b)
		if err != nil {
			core.ErrorDialog(ctx, err)
			return nil, err
		}
		bstr = text.BytesToLineStrings(fb, false) // don't add new lines
	}
	fb, err := repo.FileContents(file, rev_a)
	if err != nil {
		core.ErrorDialog(ctx, err)
		return nil, err
	}
	astr = text.BytesToLineStrings(fb, false) // don't add new lines
	if rev_a == "" {
		rev_a = "HEAD"
	}
	return DiffEditorDialog(ctx, "DiffVcs: "+fsx.DirAndFile(file), astr, bstr, file, file, rev_a, rev_b), nil
}

// DiffEditorDialog opens a dialog for displaying diff between two files as line-strings
func DiffEditorDialog(ctx core.Widget, title string, astr, bstr []string, afile, bfile, arev, brev string) *DiffEditor {
	d := core.NewBody("Diff editor")
	d.SetTitle(title)

	dv := NewDiffEditor(d)
	dv.SetFileA(afile).SetFileB(bfile).SetRevisionA(arev).SetRevisionB(brev)
	dv.DiffStrings(astr, bstr)
	d.AddAppBar(dv.MakeToolbar)
	d.NewWindow().SetContext(ctx).SetNewWindow(true).Run()
	return dv
}

// TextDialog opens a dialog for displaying text string
func TextDialog(ctx core.Widget, title, text string) *Editor {
	d := core.NewBody().AddTitle(title)
	ed := NewEditor(d)
	ed.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	ed.Buffer.SetText([]byte(text))
	d.AddBottomBar(func(parent core.Widget) {
		core.NewButton(parent).SetText("Copy to clipboard").SetIcon(icons.ContentCopy).
			OnClick(func(e events.Event) {
				d.Clipboard().Write(mimedata.NewText(text))
			})
		d.AddOK(parent)
	})
	d.RunWindowDialog(ctx)
	return ed
}

// DiffEditor presents two side-by-side [Editor]s showing the differences
// between two files (represented as lines of strings).
type DiffEditor struct {
	core.Frame

	// first file name being compared
	FileA string

	// second file name being compared
	FileB string

	// revision for first file, if relevant
	RevisionA string

	// revision for second file, if relevant
	RevisionB string

	// [Buffer] for A showing the aligned edit view
	bufferA *Buffer

	// [Buffer] for B showing the aligned edit view
	bufferB *Buffer

	// aligned diffs records diff for aligned lines
	alignD text.Diffs

	// diffs applied
	diffs text.DiffSelected

	inInputEvent bool
}

func (dv *DiffEditor) Init() {
	dv.Frame.Init()
	dv.bufferA = NewBuffer()
	dv.bufferB = NewBuffer()
	dv.bufferA.Options.LineNumbers = true
	dv.bufferB.Options.LineNumbers = true

	dv.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	f := func(name string, buf *Buffer) {
		tree.AddChildAt(dv, name, func(w *DiffTextEditor) {
			w.SetBuffer(buf)
			w.SetReadOnly(true)
			w.Styler(func(s *styles.Style) {
				s.Min.X.Ch(80)
				s.Min.Y.Em(40)
			})
			w.On(events.Scroll, func(e events.Event) {
				dv.syncEditors(events.Scroll, e, name)
			})
			w.On(events.Input, func(e events.Event) {
				dv.syncEditors(events.Input, e, name)
			})
		})
	}
	f("text-a", dv.bufferA)
	f("text-b", dv.bufferB)
}

func (dv *DiffEditor) updateToolbar() {
	tb := dv.Scene.GetTopAppBar()
	if tb == nil {
		return
	}
	tb.Restyle()
}

// setFilenames sets the filenames and updates markup accordingly.
// Called in DiffStrings
func (dv *DiffEditor) setFilenames() {
	dv.bufferA.SetFilename(dv.FileA)
	dv.bufferB.SetFilename(dv.FileB)
	dv.bufferA.Stat()
	dv.bufferB.Stat()
}

// syncEditors synchronizes the text [Editor] scrolling and cursor positions
func (dv *DiffEditor) syncEditors(typ events.Types, e events.Event, name string) {
	tva, tvb := dv.textEditors()
	me, other := tva, tvb
	if name == "text-b" {
		me, other = tvb, tva
	}
	switch typ {
	case events.Scroll:
		other.Geom.Scroll.Y = me.Geom.Scroll.Y
		other.ScrollUpdateFromGeom(math32.Y)
	case events.Input:
		if dv.inInputEvent {
			return
		}
		dv.inInputEvent = true
		other.SetCursorShow(me.CursorPos)
		dv.inInputEvent = false
	}
}

// nextDiff moves to next diff region
func (dv *DiffEditor) nextDiff(ab int) bool {
	tva, tvb := dv.textEditors()
	tv := tva
	if ab == 1 {
		tv = tvb
	}
	nd := len(dv.alignD)
	curLn := tv.CursorPos.Ln
	di, df := dv.alignD.DiffForLine(curLn)
	if di < 0 {
		return false
	}
	for {
		di++
		if di >= nd {
			return false
		}
		df = dv.alignD[di]
		if df.Tag != 'e' {
			break
		}
	}
	tva.SetCursorTarget(lexer.Pos{Ln: df.I1})
	return true
}

// prevDiff moves to previous diff region
func (dv *DiffEditor) prevDiff(ab int) bool {
	tva, tvb := dv.textEditors()
	tv := tva
	if ab == 1 {
		tv = tvb
	}
	curLn := tv.CursorPos.Ln
	di, df := dv.alignD.DiffForLine(curLn)
	if di < 0 {
		return false
	}
	for {
		di--
		if di < 0 {
			return false
		}
		df = dv.alignD[di]
		if df.Tag != 'e' {
			break
		}
	}
	tva.SetCursorTarget(lexer.Pos{Ln: df.I1})
	return true
}

// saveAs saves A or B edits into given file.
// It checks for an existing file, prompts to overwrite or not.
func (dv *DiffEditor) saveAs(ab bool, filename core.Filename) {
	if !errors.Log1(fsx.FileExists(string(filename))) {
		dv.saveFile(ab, filename)
	} else {
		d := core.NewBody().AddTitle("File Exists, Overwrite?").
			AddText(fmt.Sprintf("File already exists, overwrite?  File: %v", filename))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).OnClick(func(e events.Event) {
				dv.saveFile(ab, filename)
			})
		})
		d.RunDialog(dv)
	}
}

// saveFile writes A or B edits to file, with no prompting, etc
func (dv *DiffEditor) saveFile(ab bool, filename core.Filename) error {
	var txt string
	if ab {
		txt = strings.Join(dv.diffs.B.Edit, "\n")
	} else {
		txt = strings.Join(dv.diffs.A.Edit, "\n")
	}
	err := os.WriteFile(string(filename), []byte(txt), 0644)
	if err != nil {
		core.ErrorSnackbar(dv, err)
		slog.Error(err.Error())
	}
	return err
}

// saveFileA saves the current state of file A to given filename
func (dv *DiffEditor) saveFileA(fname core.Filename) { //types:add
	dv.saveAs(false, fname)
	dv.updateToolbar()
}

// saveFileB saves the current state of file B to given filename
func (dv *DiffEditor) saveFileB(fname core.Filename) { //types:add
	dv.saveAs(true, fname)
	dv.updateToolbar()
}

// DiffStrings computes differences between two lines-of-strings and displays in
// DiffEditor.
func (dv *DiffEditor) DiffStrings(astr, bstr []string) {
	dv.setFilenames()
	dv.diffs.SetStringLines(astr, bstr)

	dv.bufferA.LineColors = nil
	dv.bufferB.LineColors = nil
	del := colors.Scheme.Error.Base
	ins := colors.Scheme.Success.Base
	chg := colors.Scheme.Primary.Base

	nd := len(dv.diffs.Diffs)
	dv.alignD = make(text.Diffs, nd)
	var ab, bb [][]byte
	absln := 0
	bspc := []byte(" ")
	for i, df := range dv.diffs.Diffs {
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
			dv.alignD[i] = ad
			for i := 0; i < mx; i++ {
				dv.bufferA.SetLineColor(absln+i, chg)
				dv.bufferB.SetLineColor(absln+i, chg)
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
			dv.alignD[i] = ad
			for i := 0; i < di; i++ {
				dv.bufferA.SetLineColor(absln+i, ins)
				dv.bufferB.SetLineColor(absln+i, del)
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
			dv.alignD[i] = ad
			for i := 0; i < dj; i++ {
				dv.bufferA.SetLineColor(absln+i, del)
				dv.bufferB.SetLineColor(absln+i, ins)
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
			dv.alignD[i] = ad
			for i := 0; i < di; i++ {
				ab = append(ab, []byte(astr[df.I1+i]))
				bb = append(bb, []byte(bstr[df.J1+i]))
			}
			absln += di
		}
	}
	dv.bufferA.SetTextLines(ab) // don't copy
	dv.bufferB.SetTextLines(bb) // don't copy
	dv.tagWordDiffs()
	dv.bufferA.ReMarkup()
	dv.bufferB.ReMarkup()
}

// tagWordDiffs goes through replace diffs and tags differences at the
// word level between the two regions.
func (dv *DiffEditor) tagWordDiffs() {
	for _, df := range dv.alignD {
		if df.Tag != 'r' {
			continue
		}
		di := df.I2 - df.I1
		dj := df.J2 - df.J1
		mx := max(di, dj)
		stln := df.I1
		for i := 0; i < mx; i++ {
			ln := stln + i
			ra := dv.bufferA.Line(ln)
			rb := dv.bufferB.Line(ln)
			lna := lexer.RuneFields(ra)
			lnb := lexer.RuneFields(rb)
			fla := lna.RuneStrings(ra)
			flb := lnb.RuneStrings(rb)
			nab := max(len(fla), len(flb))
			ldif := text.DiffLines(fla, flb)
			ndif := len(ldif)
			if nab > 25 && ndif > nab/2 { // more than half of big diff -- skip
				continue
			}
			for _, ld := range ldif {
				switch ld.Tag {
				case 'r':
					sla := lna[ld.I1]
					ela := lna[ld.I2-1]
					dv.bufferA.AddTag(ln, sla.St, ela.Ed, token.TextStyleError)
					slb := lnb[ld.J1]
					elb := lnb[ld.J2-1]
					dv.bufferB.AddTag(ln, slb.St, elb.Ed, token.TextStyleError)
				case 'd':
					sla := lna[ld.I1]
					ela := lna[ld.I2-1]
					dv.bufferA.AddTag(ln, sla.St, ela.Ed, token.TextStyleDeleted)
				case 'i':
					slb := lnb[ld.J1]
					elb := lnb[ld.J2-1]
					dv.bufferB.AddTag(ln, slb.St, elb.Ed, token.TextStyleDeleted)
				}
			}
		}
	}
}

// applyDiff applies change from the other buffer to the buffer for given file
// name, from diff that includes given line.
func (dv *DiffEditor) applyDiff(ab int, line int) bool {
	tva, tvb := dv.textEditors()
	tv := tva
	if ab == 1 {
		tv = tvb
	}
	if line < 0 {
		line = tv.CursorPos.Ln
	}
	di, df := dv.alignD.DiffForLine(line)
	if di < 0 || df.Tag == 'e' {
		return false
	}

	if ab == 0 {
		dv.bufferA.Undos.Off = false
		// srcLen := len(dv.BufB.Lines[df.J2])
		spos := lexer.Pos{Ln: df.I1, Ch: 0}
		epos := lexer.Pos{Ln: df.I2, Ch: 0}
		src := dv.bufferB.Region(spos, epos)
		dv.bufferA.DeleteText(spos, epos, true)
		dv.bufferA.insertText(spos, src.ToBytes(), true) // we always just copy, is blank for delete..
		dv.diffs.BtoA(di)
	} else {
		dv.bufferB.Undos.Off = false
		spos := lexer.Pos{Ln: df.J1, Ch: 0}
		epos := lexer.Pos{Ln: df.J2, Ch: 0}
		src := dv.bufferA.Region(spos, epos)
		dv.bufferB.DeleteText(spos, epos, true)
		dv.bufferB.insertText(spos, src.ToBytes(), true)
		dv.diffs.AtoB(di)
	}
	dv.updateToolbar()
	return true
}

// undoDiff undoes last applied change, if any.
func (dv *DiffEditor) undoDiff(ab int) error {
	tva, tvb := dv.textEditors()
	if ab == 1 {
		if !dv.diffs.B.Undo() {
			err := errors.New("No more edits to undo")
			core.ErrorSnackbar(dv, err)
			return err
		}
		tvb.undo()
	} else {
		if !dv.diffs.A.Undo() {
			err := errors.New("No more edits to undo")
			core.ErrorSnackbar(dv, err)
			return err
		}
		tva.undo()
	}
	return nil
}

func (dv *DiffEditor) MakeToolbar(p *tree.Plan) {
	txta := "A: " + fsx.DirAndFile(dv.FileA)
	if dv.RevisionA != "" {
		txta += ": " + dv.RevisionA
	}
	tree.Add(p, func(w *core.Text) {
		w.SetText(txta)
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Next").SetIcon(icons.KeyboardArrowDown).SetTooltip("move down to next diff region")
		w.OnClick(func(e events.Event) {
			dv.nextDiff(0)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(len(dv.alignD) <= 1, states.Disabled)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Prev").SetIcon(icons.KeyboardArrowUp).SetTooltip("move up to previous diff region")
		w.OnClick(func(e events.Event) {
			dv.prevDiff(0)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(len(dv.alignD) <= 1, states.Disabled)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("A &lt;- B").SetIcon(icons.ContentCopy).SetTooltip("for current diff region, apply change from corresponding version in B, and move to next diff")
		w.OnClick(func(e events.Event) {
			dv.applyDiff(0, -1)
			dv.nextDiff(0)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(len(dv.alignD) <= 1, states.Disabled)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Undo").SetIcon(icons.Undo).SetTooltip("undo last diff apply action (A &lt;- B)")
		w.OnClick(func(e events.Event) {
			dv.undoDiff(0)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(!dv.bufferA.Changed, states.Disabled)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Save").SetIcon(icons.Save).SetTooltip("save edited version of file with the given; prompts for filename")
		w.OnClick(func(e events.Event) {
			fb := core.NewSoloFuncButton(w).SetFunc(dv.saveFileA)
			fb.Args[0].SetValue(core.Filename(dv.FileA))
			fb.CallFunc()
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(!dv.bufferA.Changed, states.Disabled)
		})
	})

	tree.Add(p, func(w *core.Separator) {})

	txtb := "B: " + fsx.DirAndFile(dv.FileB)
	if dv.RevisionB != "" {
		txtb += ": " + dv.RevisionB
	}
	tree.Add(p, func(w *core.Text) {
		w.SetText(txtb)
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Next").SetIcon(icons.KeyboardArrowDown).SetTooltip("move down to next diff region")
		w.OnClick(func(e events.Event) {
			dv.nextDiff(1)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(len(dv.alignD) <= 1, states.Disabled)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Prev").SetIcon(icons.KeyboardArrowUp).SetTooltip("move up to previous diff region")
		w.OnClick(func(e events.Event) {
			dv.prevDiff(1)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(len(dv.alignD) <= 1, states.Disabled)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("A -&gt; B").SetIcon(icons.ContentCopy).SetTooltip("for current diff region, apply change from corresponding version in A, and move to next diff")
		w.OnClick(func(e events.Event) {
			dv.applyDiff(1, -1)
			dv.nextDiff(1)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(len(dv.alignD) <= 1, states.Disabled)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Undo").SetIcon(icons.Undo).SetTooltip("undo last diff apply action (A -&gt; B)")
		w.OnClick(func(e events.Event) {
			dv.undoDiff(1)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(!dv.bufferB.Changed, states.Disabled)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Save").SetIcon(icons.Save).SetTooltip("save edited version of file -- prompts for filename -- this will convert file back to its original form (removing side-by-side alignment) and end the diff editing function")
		w.OnClick(func(e events.Event) {
			fb := core.NewSoloFuncButton(w).SetFunc(dv.saveFileB)
			fb.Args[0].SetValue(core.Filename(dv.FileB))
			fb.CallFunc()
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(!dv.bufferB.Changed, states.Disabled)
		})
	})
}

func (dv *DiffEditor) textEditors() (*DiffTextEditor, *DiffTextEditor) {
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

func (ed *DiffTextEditor) Init() {
	ed.Editor.Init()
	ed.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	ed.OnDoubleClick(func(e events.Event) {
		pt := ed.PointToRelPos(e.Pos())
		if pt.X >= 0 && pt.X < int(ed.LineNumberOffset) {
			newPos := ed.PixelToCursor(pt)
			ln := newPos.Ln
			dv := ed.diffEditor()
			if dv != nil && ed.Buffer != nil {
				if ed.Name == "text-a" {
					dv.applyDiff(0, ln)
				} else {
					dv.applyDiff(1, ln)
				}
			}
			e.SetHandled()
			return
		}
	})
}

func (ed *DiffTextEditor) diffEditor() *DiffEditor {
	return tree.ParentByType[*DiffEditor](ed)
}
