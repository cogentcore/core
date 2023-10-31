// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/glop/dirs"
	"goki.dev/goosi/events"
	"goki.dev/goosi/mimedata"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/vci/v2"
)

// VCSLogView is a view of the variables
type VCSLogView struct {
	gi.Layout

	// current log
	Log vci.Log

	// file that this is a log of -- if blank then it is entire repository
	File string

	// date expression for how long ago to include log entries from
	Since string

	// version control system repository
	Repo vci.Repo `json:"-" xml:"-" copy:"-"`

	// revision A -- defaults to HEAD
	RevA string `set:"-"`

	// revision B -- blank means current working copy
	RevB string `set:"-"`

	// double-click will set the A revision -- else B
	SetA bool
}

func (lv *VCSLogView) OnInit() {
	lv.Style(func(s *styles.Style) {
		s.SetStretchMax()
	})
	lv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(lv) {
		case "a-tf", "b-tf":
			w.Style(func(s *styles.Style) {
				s.Width.Em(12)
			})
		}
	})
}

// ConfigRepo configures to given repo, log and file (file could be empty)
func (lv *VCSLogView) ConfigRepo(repo vci.Repo, lg vci.Log, file, since string) {
	lv.Repo = repo
	lv.Log = lg
	lv.File = file
	lv.Since = since
	lv.Lay = gi.LayoutVert
	config := ki.Config{}
	config.Add(gi.ToolbarType, "toolbar")
	config.Add(TableViewType, "log")
	mods, updt := lv.ConfigChildren(config)
	tv := lv.TableView()
	if mods {
		lv.RevA = "HEAD"
		lv.RevB = ""
		lv.SetA = true
		lv.ConfigToolbar()
		tv.OnDoubleClick(func(e events.Event) {
			idx := tv.CurIdx
			if idx >= 0 && idx < len(lv.Log) {
				cmt := lv.Log[idx]
				if lv.File != "" {
					if lv.SetA {
						lv.SetRevA(cmt.Rev)
					} else {
						lv.SetRevB(cmt.Rev)
					}
					lv.ToggleRev()
				}
				cinfo, err := lv.Repo.CommitDesc(cmt.Rev, false)
				if err == nil {
					d := gi.NewDialog(lv).Title("Commit Info: " + cmt.Rev).FullWindow(true)
					buf := texteditor.NewBuf()
					buf.Filename = gi.FileName(lv.File)
					buf.Opts.LineNos = true
					buf.Stat()
					texteditor.NewEditor(d).SetBuf(buf)
					buf.SetText(cinfo)
					gi.NewButton(d.Buttons()).SetText("Copy to clipboard").SetIcon(icons.ContentCopy).
						OnClick(func(e events.Event) {
							d.EventMgr.ClipBoard().Write(mimedata.NewTextBytes(cinfo))
						})
					d.Ok().Run()
				}
			}
		})
	} else {
		updt = lv.UpdateStart()
	}
	tv.SetState(true, states.ReadOnly)
	tv.SetSlice(&lv.Log)
	lv.UpdateEndLayout(updt)
}

// SetRevA sets the RevA to use
func (lv *VCSLogView) SetRevA(rev string) {
	lv.RevA = rev
	tb := lv.Toolbar()
	tfi := tb.ChildByName("a-tf", 2)
	if tfi == nil {
		return
	}
	tfi.(*gi.TextField).SetText(rev)
}

// SetRevB sets the RevB to use
func (lv *VCSLogView) SetRevB(rev string) {
	lv.RevB = rev
	tb := lv.Toolbar()
	tfi := tb.ChildByName("b-tf", 2)
	if tfi == nil {
		return
	}
	tfi.(*gi.TextField).SetText(rev)
}

// ToggleRev switches the active revision to set
func (lv *VCSLogView) ToggleRev() {
	tb := lv.Toolbar()
	updt := tb.UpdateStart()
	cba := tb.ChildByName("a-rev", 2).(*gi.Switch)
	cbb := tb.ChildByName("b-rev", 2).(*gi.Switch)
	lv.SetA = !lv.SetA
	cba.SetState(lv.SetA, states.Checked)
	cbb.SetState(!lv.SetA, states.Checked)
	tb.UpdateEnd(updt)
}

// Toolbar returns the toolbar
func (lv *VCSLogView) Toolbar() *gi.Toolbar {
	return lv.ChildByName("toolbar", 0).(*gi.Toolbar)
}

// TableView returns the tableview
func (lv *VCSLogView) TableView() *TableView {
	return lv.ChildByName("log", 1).(*TableView)
}

// ConfigToolbar
func (lv *VCSLogView) ConfigToolbar() {
	tb := lv.Toolbar()
	if lv.File != "" {
		gi.NewLabel(tb, "fl", "File: "+dirs.DirAndFile(lv.File))
		gi.NewSeparator(tb, "flsep")
		cba := gi.NewSwitch(tb, "a-rev")
		cba.SetText("A Rev: ")
		cba.Tooltip = "If selected, double-clicking in log will set this A Revision to use for Diff"
		cba.SetState(true, states.Checked)
		tfa := gi.NewTextField(tb, "a-tf")
		tfa.SetText(lv.RevA)
		tfa.OnChange(func(e events.Event) {
			lv.RevA = tfa.Text()
		})
		gi.NewSeparator(tb, "absep")
		cbb := gi.NewSwitch(tb, "b-rev")
		cbb.SetText("B Rev: ")
		cbb.Tooltip = "If selected, double-clicking in log will set this B Revision to use for Diff"
		tfb := gi.NewTextField(tb, "b-tf")
		tfb.SetText(lv.RevB)
		tfb.OnChange(func(e events.Event) {
			lv.RevB = tfb.Text()
		})
		gi.NewSeparator(tb, "dsep")
		gi.NewButton(tb, "diff").SetText("Diff").SetIcon(icons.Difference).SetTooltip("Show the diffs between two revisions -- if blank, A is current HEAD, and B is current working copy").
			OnClick(func(e events.Event) {
				// TOOD: add this back
				// DiffViewDialogFromRevs(lv.Sc, lv.Repo, lv.File, nil, lv.RevA, lv.RevB)
			})
		cba.OnClick(func(e events.Event) {
			lv.SetA = cba.StateIs(states.Checked)
		})
		cbb.OnClick(func(e events.Event) {
			lv.SetA = !cbb.StateIs(states.Checked)
			// cba.SetState(lv.SetA, states.Checked)
		})
	}

}

// VCSLogViewDialog returns a VCS Log View for given repo, log and file (file could be empty)
func VCSLogViewDialog(ctx gi.Widget, repo vci.Repo, lg vci.Log, file, since string) *gi.Dialog {
	title := "VCS Log: "
	if file == "" {
		title += "All files"
	} else {
		title += dirs.DirAndFile(file)
	}
	if since != "" {
		title += " since: " + since
	}
	d := gi.NewDialog(ctx).Title(title).NewWindow(true)

	lv := NewVCSLogView(d, "vcslog")
	lv.ConfigRepo(repo, lg, file, since)

	return d
}
