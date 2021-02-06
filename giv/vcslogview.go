// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/vci"
)

// VCSLogView is a view of the variables
type VCSLogView struct {
	gi.Layout
	Log   vci.Log  `desc:"current log"`
	File  string   `desc:"file that this is a log of -- if blank then it is entire repository"`
	Since string   `desc:"date expression for how long ago to include log entries from"`
	Repo  vci.Repo `json:"-" xml:"-" copy:"-" desc:"version control system repository"`
	RevA  string   `desc:"revision A -- defaults to HEAD"`
	RevB  string   `desc:"revision B -- blank means current working copy"`
	SetA  bool     `desc:"double-click will set the A revision -- else B"`
}

var KiT_VCSLogView = kit.Types.AddType(&VCSLogView{}, VCSLogViewProps)

// Config configures to given repo, log and file (file could be empty)
func (lv *VCSLogView) Config(repo vci.Repo, lg vci.Log, file, since string) {
	lv.Repo = repo
	lv.Log = lg
	lv.File = file
	lv.Since = since
	lv.Lay = gi.LayoutVert
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(KiT_TableView, "log")
	mods, updt := lv.ConfigChildren(config)
	tv := lv.TableView()
	if mods {
		lv.RevA = "HEAD"
		lv.RevB = ""
		lv.SetA = true
		lv.ConfigToolBar()
		tv.SliceViewSig.Connect(lv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(SliceViewDoubleClicked) {
				idx := data.(int)
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
						TextViewDialog(lv.ViewportSafe(), cinfo, DlgOpts{Title: "Commit Info: " + cmt.Rev, Ok: true})
					}
				}
			}
		})
	} else {
		updt = lv.UpdateStart()
	}
	tv.SetStretchMax()
	tv.SetInactive()
	tv.SetSlice(&lv.Log)
	lv.UpdateEnd(updt)
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
	cba := tb.ChildByName("a-rev", 2).(*gi.CheckBox)
	cbb := tb.ChildByName("b-rev", 2).(*gi.CheckBox)
	lv.SetA = !lv.SetA
	cba.SetChecked(lv.SetA)
	cbb.SetChecked(!lv.SetA)
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
		gi.AddNewLabel(tb, "fl", "File: "+DirAndFile(lv.File))
		tb.AddSeparator("flsep")
		cba := gi.AddNewCheckBox(tb, "a-rev")
		cba.SetText("A Rev: ")
		cba.Tooltip = "If selected, double-clicking in log will set this A Revision to use for Diff"
		cba.SetChecked(true)
		tfa := gi.AddNewTextField(tb, "a-tf")
		tfa.SetProp("width", "12em")
		tfa.SetText(lv.RevA)
		tfa.TextFieldSig.Connect(lv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
				lv.RevA = tfa.Text()
			}
		})
		tb.AddSeparator("absep")
		cbb := gi.AddNewCheckBox(tb, "b-rev")
		cbb.SetText("B Rev: ")
		cbb.Tooltip = "If selected, double-clicking in log will set this B Revision to use for Diff"
		tfb := gi.AddNewTextField(tb, "b-tf")
		tfb.SetProp("width", "12em")
		tfb.SetText(lv.RevB)
		tfb.TextFieldSig.Connect(lv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
				lv.RevB = tfb.Text()
			}
		})
		tb.AddSeparator("dsep")
		tb.AddAction(gi.ActOpts{Label: "Diff", Icon: "file-sheet", Tooltip: "Show the diffs between two revisions -- if blank, A is current HEAD, and B is current working copy"}, lv.This(),
			func(recv, send ki.Ki, sig int64, data interface{}) {
				lvv := recv.Embed(KiT_VCSLogView).(*VCSLogView)
				DiffViewDialogFromRevs(lvv.ViewportSafe(), lvv.Repo, lvv.File, nil, lvv.RevA, lvv.RevB)
			})

		cba.ButtonSig.Connect(lv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.ButtonToggled) {
				lv.SetA = cba.IsChecked()
				cbb.SetChecked(!lv.SetA)
				cbb.UpdateSig()
			}
		})
		cbb.ButtonSig.Connect(lv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.ButtonToggled) {
				lv.SetA = !cbb.IsChecked()
				cba.SetChecked(lv.SetA)
				cba.UpdateSig()
			}
		})
	}

}

// VCSLogViewProps are style properties for DebugView
var VCSLogViewProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"max-width":     -1,
	"max-height":    -1,
}

// VCSLogViewDialog opens a VCS Log View for given repo, log and file (file could be empty)
func VCSLogViewDialog(repo vci.Repo, lg vci.Log, file, since string) *gi.Dialog {
	title := "VCS Log: "
	if file == "" {
		title += "All files"
	} else {
		title += DirAndFile(file)
	}
	if since != "" {
		title += " since: " + since
	}
	dlg := gi.NewStdDialog(gi.DlgOpts{Title: title}, gi.NoOk, gi.NoCancel)
	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	lv := frame.InsertNewChild(KiT_VCSLogView, prIdx+1, "vcslog").(*VCSLogView)
	lv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	lv.Config(repo, lg, file, since)

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, nil, nil)
	return dlg
}
