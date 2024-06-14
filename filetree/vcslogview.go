// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"log/slog"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/diffbrowser"
)

// VCSLogView is a view of the VCS log data
type VCSLogView struct {
	core.Frame

	// current log
	Log vcs.Log

	// file that this is a log of -- if blank then it is entire repository
	File string

	// date expression for how long ago to include log entries from
	Since string

	// version control system repository
	Repo vcs.Repo `json:"-" xml:"-" copier:"-"`

	// revision A -- defaults to HEAD
	RevA string `set:"-"`

	// revision B -- blank means current working copy
	RevB string `set:"-"`

	// double-click will set the A revision -- else B
	SetA bool
}

func (lv *VCSLogView) Init() {
	lv.Frame.Init()
	lv.RevA = "HEAD"
	lv.RevB = ""
	lv.SetA = true
	lv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	core.AddChildAt(lv, "toolbar", func(w *core.Toolbar) {
		w.Maker(lv.MakeToolbar)
	})
	core.AddChildAt(lv, "log", func(w *core.Table) {
		w.SetReadOnly(true)
		w.SetSlice(&lv.Log)
		w.AddContextMenu(func(m *core.Scene) {
			core.NewButton(m).SetText("Set Revision A").
				SetTooltip("Set Buffer A's revision to this").
				OnClick(func(e events.Event) {
					cmt := lv.Log[w.SelectedIndex]
					lv.SetRevA(cmt.Rev)
				})
			core.NewButton(m).SetText("Set Revision B").
				SetTooltip("Set Buffer B's revision to this").
				OnClick(func(e events.Event) {
					cmt := lv.Log[w.SelectedIndex]
					lv.SetRevB(cmt.Rev)
				})
			core.NewButton(m).SetText("Copy Revision ID").
				SetTooltip("Copies the revision number / hash for this").
				OnClick(func(e events.Event) {
					cmt := lv.Log[w.SelectedIndex]
					w.Clipboard().Write(mimedata.NewText(cmt.Rev))
				})
			core.NewButton(m).SetText("View Revision").
				SetTooltip("Views the file at this revision").
				OnClick(func(e events.Event) {
					cmt := lv.Log[w.SelectedIndex]
					FileAtRevDialog(lv, lv.Repo, lv.File, cmt.Rev)
				})
			core.NewButton(m).SetText("Checkout Revision").
				SetTooltip("Checks out this revision").
				OnClick(func(e events.Event) {
					cmt := lv.Log[w.SelectedIndex]
					errors.Log(lv.Repo.UpdateVersion(cmt.Rev))
				})
		})
		w.OnSelect(func(e events.Event) {
			idx := w.SelectedIndex
			if idx < 0 || idx >= len(lv.Log) {
				return
			}
			cmt := lv.Log[idx]
			if lv.SetA {
				lv.SetRevA(cmt.Rev)
			} else {
				lv.SetRevB(cmt.Rev)
			}
			lv.ToggleRev()
		})
		w.OnDoubleClick(func(e events.Event) {
			idx := w.SelectedIndex
			if idx < 0 || idx >= len(lv.Log) {
				return
			}
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
			if err != nil {
				slog.Error(err.Error())
				return
			}
			d := core.NewBody().AddTitle("Commit Info: " + cmt.Rev)
			buf := texteditor.NewBuffer()
			buf.Filename = core.Filename(lv.File)
			buf.Options.LineNumbers = true
			buf.Stat()
			texteditor.NewEditor(d).SetBuffer(buf)
			buf.SetText(cinfo)
			d.AddBottomBar(func(parent core.Widget) {
				core.NewButton(parent).SetText("Copy to clipboard").SetIcon(icons.ContentCopy).
					OnClick(func(e events.Event) {
						d.Clipboard().Write(mimedata.NewTextBytes(cinfo))
					})
				d.AddOK(parent)
			})
			d.RunFullDialog(lv)
		})
	})
}

// SetRevA sets the revision to use for buffer A
func (lv *VCSLogView) SetRevA(rev string) {
	lv.RevA = rev
	tb := lv.Toolbar()
	tfi := tb.ChildByName("a-tf", 2)
	if tfi == nil {
		return
	}
	tfi.(*core.TextField).SetText(rev)
}

// SetRevB sets the revision to use for buffer B
func (lv *VCSLogView) SetRevB(rev string) {
	lv.RevB = rev
	tb := lv.Toolbar()
	tfi := tb.ChildByName("b-tf", 2)
	if tfi == nil {
		return
	}
	tfi.(*core.TextField).SetText(rev)
}

// ToggleRev switches the active revision to set
func (lv *VCSLogView) ToggleRev() {
	tb := lv.Toolbar()
	cba := tb.ChildByName("a-rev", 2).(*core.Switch)
	cbb := tb.ChildByName("b-rev", 2).(*core.Switch)
	lv.SetA = !lv.SetA
	cba.SetState(lv.SetA, states.Checked)
	cbb.SetState(!lv.SetA, states.Checked)
	cbb.NeedsRender()
	cba.NeedsRender()
}

// Toolbar returns the toolbar
func (lv *VCSLogView) Toolbar() *core.Toolbar {
	return lv.ChildByName("toolbar", 0).(*core.Toolbar)
}

// Table returns the table
func (lv *VCSLogView) Table() *core.Table {
	return lv.ChildByName("log", 1).(*core.Table)
}

func (lv *VCSLogView) MakeToolbar(p *core.Plan) {
	core.Add(p, func(w *core.Text) {
		w.SetText("File: " + dirs.DirAndFile(lv.File))
	})

	core.AddAt(p, "a-rev", func(w *core.Switch) {
		w.SetText("A Rev: ")
		w.SetTooltip("If selected, clicking in log will set this A Revision to use for Diff")
		w.SetState(true, states.Checked)
		w.OnClick(func(e events.Event) {
			lv.SetA = w.IsChecked()
			cbb := w.Parent.AsTree().ChildByName("b-rev", 2).(*core.Switch)
			cbb.SetState(!lv.SetA, states.Checked)
			cbb.NeedsRender()
		})
	})
	core.AddAt(p, "a-tf", func(w *core.TextField) {
		w.SetText(lv.RevA)
		w.SetTooltip("A revision: typically this is the older, base revision to compare")
		w.OnChange(func(e events.Event) {
			lv.RevA = w.Text()
		})
	})
	core.Add(p, func(w *core.Button) {
		w.SetText("View A").SetIcon(icons.Document).
			SetTooltip("View file at revision A").
			OnClick(func(e events.Event) {
				FileAtRevDialog(lv, lv.Repo, lv.File, lv.RevA)
			})
	})

	core.Add(p, func(w *core.Separator) {})

	core.AddAt(p, "b-rev", func(w *core.Switch) {
		w.SetText("B Rev: ")
		w.SetTooltip("If selected, clicking in log will set this B Revision to use for Diff")
		w.OnClick(func(e events.Event) {
			lv.SetA = !w.IsChecked()
			cba := w.Parent.AsTree().ChildByName("a-rev", 2).(*core.Switch)
			cba.SetState(lv.SetA, states.Checked)
			cba.NeedsRender()
		})
	})

	core.AddAt(p, "b-tf", func(w *core.TextField) {
		w.SetText(lv.RevB)
		w.SetTooltip("B revision: typically this is the newer revision to compare.  Leave blank for the current working directory.")
		w.OnChange(func(e events.Event) {
			lv.RevB = w.Text()
		})
	})
	core.Add(p, func(w *core.Button) {
		w.SetText("View B").SetIcon(icons.Document).
			SetTooltip("View file at revision B").
			OnClick(func(e events.Event) {
				FileAtRevDialog(lv, lv.Repo, lv.File, lv.RevB)
			})
	})

	core.Add(p, func(w *core.Separator) {})

	core.Add(p, func(w *core.Button) {
		w.SetText("Diff").SetIcon(icons.Difference).
			SetTooltip("Show the diffs between two revisions; if blank, A is current HEAD, and B is current working copy").
			OnClick(func(e events.Event) {
				if lv.File == "" {
					diffbrowser.NewDiffBrowserVCS(lv.Repo, lv.RevA, lv.RevB)
				} else {
					texteditor.DiffEditorDialogFromRevs(lv, lv.Repo, lv.File, nil, lv.RevA, lv.RevB)
				}
			})
	})
}

// VCSLogViewDialog returns a VCS Log View for given repo, log and file (file could be empty)
func VCSLogViewDialog(ctx core.Widget, repo vcs.Repo, lg vcs.Log, file, since string) *core.Body {
	title := "VCS Log: "
	if file == "" {
		title += "All files"
	} else {
		title += dirs.DirAndFile(file)
	}
	if since != "" {
		title += " since: " + since
	}
	d := core.NewBody().AddTitle(title)
	lv := NewVCSLogView(d)
	lv.SetRepo(repo).SetLog(lg).SetFile(file).SetSince(since)
	d.RunWindowDialog(ctx)
	return d
}

// FileAtRevDialog Shows a file at a given revision in a new dialog window
func FileAtRevDialog(ctx core.Widget, repo vcs.Repo, file, rev string) *core.Body {
	fb, err := repo.FileContents(file, rev)
	if err != nil {
		core.ErrorDialog(ctx, err)
		return nil
	}
	if rev == "" {
		rev = "HEAD"
	}
	title := "File at VCS Revision: " + dirs.DirAndFile(file) + "@" + rev
	d := core.NewBody().AddTitle(title)

	tb := texteditor.NewBuffer().SetText(fb).SetFilename(file) // file is key for getting lang
	texteditor.NewEditor(d).SetBuffer(tb).SetReadOnly(true)
	d.RunWindowDialog(ctx)
	tb.StartDelayedReMarkup() // update markup
	return d
}
