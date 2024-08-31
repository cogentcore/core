// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"log/slog"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/diffbrowser"
	"cogentcore.org/core/tree"
)

// VCSLog is a widget that represents VCS log data.
type VCSLog struct {
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
	revisionA string

	// revision B -- blank means current working copy
	revisionB string

	// double-click will set the A revision -- else B
	setA bool

	arev, brev *core.Switch
	atf, btf   *core.TextField
}

func (lv *VCSLog) Init() {
	lv.Frame.Init()
	lv.revisionA = "HEAD"
	lv.revisionB = ""
	lv.setA = true
	lv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	tree.AddChildAt(lv, "toolbar", func(w *core.Toolbar) {
		w.Maker(lv.makeToolbar)
	})
	tree.AddChildAt(lv, "log", func(w *core.Table) {
		w.SetReadOnly(true)
		w.SetSlice(&lv.Log)
		w.AddContextMenu(func(m *core.Scene) {
			core.NewButton(m).SetText("Set Revision A").
				SetTooltip("Set Buffer A's revision to this").
				OnClick(func(e events.Event) {
					cmt := lv.Log[w.SelectedIndex]
					lv.setRevisionA(cmt.Rev)
				})
			core.NewButton(m).SetText("Set Revision B").
				SetTooltip("Set Buffer B's revision to this").
				OnClick(func(e events.Event) {
					cmt := lv.Log[w.SelectedIndex]
					lv.setRevisionB(cmt.Rev)
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
					fileAtRevisionDialog(lv, lv.Repo, lv.File, cmt.Rev)
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
			if lv.setA {
				lv.setRevisionA(cmt.Rev)
			} else {
				lv.setRevisionB(cmt.Rev)
			}
			lv.toggleRevision()
		})
		w.OnDoubleClick(func(e events.Event) {
			idx := w.SelectedIndex
			if idx < 0 || idx >= len(lv.Log) {
				return
			}
			cmt := lv.Log[idx]
			if lv.File != "" {
				if lv.setA {
					lv.setRevisionA(cmt.Rev)
				} else {
					lv.setRevisionB(cmt.Rev)
				}
				lv.toggleRevision()
			}
			cinfo, err := lv.Repo.CommitDesc(cmt.Rev, false)
			if err != nil {
				slog.Error(err.Error())
				return
			}
			d := core.NewBody("Commit Info: " + cmt.Rev)
			buf := texteditor.NewBuffer()
			buf.Filename = core.Filename(lv.File)
			buf.Options.LineNumbers = true
			buf.Stat()
			texteditor.NewEditor(d).SetBuffer(buf).Styler(func(s *styles.Style) {
				s.Grow.Set(1, 1)
			})
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

// setRevisionA sets the revision to use for buffer A
func (lv *VCSLog) setRevisionA(rev string) {
	lv.revisionA = rev
	lv.atf.Update()
}

// setRevisionB sets the revision to use for buffer B
func (lv *VCSLog) setRevisionB(rev string) {
	lv.revisionB = rev
	lv.btf.Update()
}

// toggleRevision switches the active revision to set
func (lv *VCSLog) toggleRevision() {
	lv.setA = !lv.setA
	lv.arev.UpdateRender()
	lv.brev.UpdateRender()
}

func (lv *VCSLog) makeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.Text) {
		w.SetText("File: " + fsx.DirAndFile(lv.File))
	})

	tree.AddAt(p, "a-rev", func(w *core.Switch) {
		lv.arev = w
		core.Bind(&lv.setA, w)
		w.SetText("A Rev: ")
		w.SetTooltip("If selected, clicking in log will set this A Revision to use for Diff")
		w.OnChange(func(e events.Event) {
			lv.brev.UpdateRender()
		})
	})
	tree.AddAt(p, "a-tf", func(w *core.TextField) {
		lv.atf = w
		core.Bind(&lv.revisionA, w)
		w.SetTooltip("A revision: typically this is the older, base revision to compare")
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("View A").SetIcon(icons.Document).
			SetTooltip("View file at revision A").
			OnClick(func(e events.Event) {
				fileAtRevisionDialog(lv, lv.Repo, lv.File, lv.revisionA)
			})
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.AddAt(p, "b-rev", func(w *core.Switch) {
		lv.brev = w
		w.SetText("B Rev: ")
		w.SetTooltip("If selected, clicking in log will set this B Revision to use for Diff")
		w.Updater(func() {
			w.SetChecked(!lv.setA)
		})
		w.OnChange(func(e events.Event) {
			lv.setA = !w.IsChecked()
			lv.arev.UpdateRender()
		})
	})

	tree.AddAt(p, "b-tf", func(w *core.TextField) {
		lv.btf = w
		core.Bind(&lv.revisionB, w)
		w.SetTooltip("B revision: typically this is the newer revision to compare.  Leave blank for the current working directory.")
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("View B").SetIcon(icons.Document).
			SetTooltip("View file at revision B").
			OnClick(func(e events.Event) {
				fileAtRevisionDialog(lv, lv.Repo, lv.File, lv.revisionB)
			})
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Diff").SetIcon(icons.Difference).
			SetTooltip("Show the diffs between two revisions; if blank, A is current HEAD, and B is current working copy").
			OnClick(func(e events.Event) {
				if lv.File == "" {
					diffbrowser.NewDiffBrowserVCS(lv.Repo, lv.revisionA, lv.revisionB)
				} else {
					texteditor.DiffEditorDialogFromRevs(lv, lv.Repo, lv.File, nil, lv.revisionA, lv.revisionB)
				}
			})
	})
}

// vcsLogDialog returns a VCS Log View for given repo, log and file (file could be empty)
func vcsLogDialog(ctx core.Widget, repo vcs.Repo, lg vcs.Log, file, since string) *core.Body {
	title := "VCS Log: "
	if file == "" {
		title += "All files"
	} else {
		title += fsx.DirAndFile(file)
	}
	if since != "" {
		title += " since: " + since
	}
	d := core.NewBody(title)
	lv := NewVCSLog(d)
	lv.SetRepo(repo).SetLog(lg).SetFile(file).SetSince(since)
	d.RunWindowDialog(ctx)
	return d
}

// fileAtRevisionDialog shows a file at a given revision in a new dialog window
func fileAtRevisionDialog(ctx core.Widget, repo vcs.Repo, file, rev string) *core.Body {
	fb, err := repo.FileContents(file, rev)
	if err != nil {
		core.ErrorDialog(ctx, err)
		return nil
	}
	if rev == "" {
		rev = "HEAD"
	}
	title := "File at VCS Revision: " + fsx.DirAndFile(file) + "@" + rev
	d := core.NewBody(title)

	tb := texteditor.NewBuffer().SetText(fb).SetFilename(file) // file is key for getting lang
	texteditor.NewEditor(d).SetBuffer(tb).SetReadOnly(true).Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	d.RunWindowDialog(ctx)
	tb.ReMarkup() // update markup
	return d
}
