// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/vci/v2"
)

// VCSLogView is a view of the variables
type VCSLogView struct {
	gi.Layout

	// current log
	Log vci.Log `desc:"current log"`

	// file that this is a log of -- if blank then it is entire repository
	File string `desc:"file that this is a log of -- if blank then it is entire repository"`

	// date expression for how long ago to include log entries from
	Since string `desc:"date expression for how long ago to include log entries from"`

	// version control system repository
	Repo vci.Repo `json:"-" xml:"-" copy:"-" desc:"version control system repository"`

	// revision A -- defaults to HEAD
	RevA string `desc:"revision A -- defaults to HEAD"`

	// revision B -- blank means current working copy
	RevB string `desc:"revision B -- blank means current working copy"`

	// double-click will set the A revision -- else B
	SetA bool `desc:"double-click will set the A revision -- else B"`
}

func (lv *VCSLogView) OnInit() {
	lv.AddStyles(func(s *styles.Style) {
		s.SetStretchMax()
	})
}

func (lv *VCSLogView) OnChildAdded(child ki.Ki) {
	w, _ := gi.AsWidget(child)
	switch w.Name() {
	case "a-tf", "b-tf":
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetEm(12)
		})
	}
}

// ConfigRepo configures to given repo, log and file (file could be empty)
func (lv *VCSLogView) ConfigRepo(repo vci.Repo, lg vci.Log, file, since string) {
	lv.Repo = repo
	lv.Log = lg
	lv.File = file
	lv.Since = since
	lv.Lay = gi.LayoutVert
	config := ki.Config{}
	config.Add(gi.ToolBarType, "toolbar")
	config.Add(TableViewType, "log")
	mods, updt := lv.ConfigChildren(config)
	tv := lv.TableView()
	if mods {
		lv.RevA = "HEAD"
		lv.RevB = ""
		lv.SetA = true
		lv.ConfigToolBar()
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
					TextViewDialog(lv, DlgOpts{Title: "Commit Info: " + cmt.Rev, Ok: true}, cinfo, nil)
				}
			}
		})
	} else {
		updt = lv.UpdateStart()
	}
	tv.SetState(true, states.Disabled)
	tv.SetSlice(&lv.Log)
	lv.UpdateEndLayout(updt)
}

// SetRevA sets the RevA to use
func (lv *VCSLogView) SetRevA(rev string) {
	lv.RevA = rev
	tb := lv.ToolBar()
	tfi := tb.ChildByName("a-tf", 2)
	if tfi == nil {
		return
	}
	tfi.(*gi.TextField).SetText(rev)
}

// SetRevB sets the RevB to use
func (lv *VCSLogView) SetRevB(rev string) {
	lv.RevB = rev
	tb := lv.ToolBar()
	tfi := tb.ChildByName("b-tf", 2)
	if tfi == nil {
		return
	}
	tfi.(*gi.TextField).SetText(rev)
}

// ToggleRev switches the active revision to set
func (lv *VCSLogView) ToggleRev() {
	tb := lv.ToolBar()
	updt := tb.UpdateStart()
	cba := tb.ChildByName("a-rev", 2).(*gi.Switch)
	cbb := tb.ChildByName("b-rev", 2).(*gi.Switch)
	lv.SetA = !lv.SetA
	cba.SetState(lv.SetA, states.Checked)
	cbb.SetState(!lv.SetA, states.Checked)
	tb.UpdateEnd(updt)
}

// ToolBar returns the toolbar
func (lv *VCSLogView) ToolBar() *gi.ToolBar {
	return lv.ChildByName("toolbar", 0).(*gi.ToolBar)
}

// TableView returns the tableview
func (lv *VCSLogView) TableView() *TableView {
	return lv.ChildByName("log", 1).(*TableView)
}

// ConfigToolBar
func (lv *VCSLogView) ConfigToolBar() {
	tb := lv.ToolBar()
	if lv.File != "" {
		gi.NewLabel(tb, "fl", "File: "+DirAndFile(lv.File))
		tb.AddSeparator("flsep")
		cba := gi.NewSwitch(tb, "a-rev")
		cba.SetText("A Rev: ")
		cba.Tooltip = "If selected, double-clicking in log will set this A Revision to use for Diff"
		cba.SetState(true, states.Checked)
		tfa := gi.NewTextField(tb, "a-tf")
		tfa.SetText(lv.RevA)
		tfa.OnChange(func(e events.Event) {
			lv.RevA = tfa.Text()
		})
		tb.AddSeparator("absep")
		cbb := gi.NewSwitch(tb, "b-rev")
		cbb.SetText("B Rev: ")
		cbb.Tooltip = "If selected, double-clicking in log will set this B Revision to use for Diff"
		tfb := gi.NewTextField(tb, "b-tf")
		tfb.SetText(lv.RevB)
		tfb.OnChange(func(e events.Event) {
			lv.RevB = tfb.Text()
		})
		tb.AddSeparator("dsep")
		tb.AddButton(gi.ActOpts{Label: "Diff", Icon: icons.Difference, Tooltip: "Show the diffs between two revisions -- if blank, A is current HEAD, and B is current working copy"}, func(act *gi.Button) {
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

// VCSLogViewDialog opens a VCS Log View for given repo, log and file (file could be empty)
func VCSLogViewDialog(ctx gi.Widget, repo vci.Repo, lg vci.Log, file, since string) *gi.Dialog {
	title := "VCS Log: "
	if file == "" {
		title += "All files"
	} else {
		title += DirAndFile(file)
	}
	if since != "" {
		title += " since: " + since
	}
	dlg := gi.NewStdDialog(ctx, gi.DlgOpts{Title: title}, nil)
	frame := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	lv := frame.InsertNewChild(VCSLogViewType, prIdx+1, "vcslog").(*VCSLogView)
	lv.ConfigRepo(repo, lg, file, since)

	return dlg
}
