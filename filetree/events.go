// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"fmt"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/vcs"
	"cogentcore.org/core/views"
)

func (fn *Node) HandleEvents() {
	fn.On(events.KeyChord, func(e events.Event) {
		kt := e.(*events.Key)
		fn.KeyInput(kt)
	})
}

func (fn *Node) KeyInput(kt events.Event) {
	if core.DebugSettings.KeyEventTrace {
		fmt.Printf("TreeView KeyInput: %v\n", fn.Path())
	}
	kf := keymap.Of(kt.KeyChord())
	selMode := events.SelectModeBits(kt.Modifiers())

	if selMode == events.SelectOne {
		if fn.SelectMode() {
			selMode = events.ExtendContinuous
		}
	}

	// first all the keys that work for ReadOnly and active
	if !fn.IsReadOnly() && !kt.IsHandled() {
		switch kf {
		case keymap.Delete:
			fn.DeleteFiles()
			kt.SetHandled()
		case keymap.Backspace:
			fn.DeleteFiles()
			kt.SetHandled()
		case keymap.Duplicate:
			fn.DuplicateFiles()
			kt.SetHandled()
		case keymap.Insert: // New File
			views.CallFunc(fn, fn.NewFile)
			kt.SetHandled()
		case keymap.InsertAfter: // New Folder
			views.CallFunc(fn, fn.NewFolder)
			kt.SetHandled()
		}
	}
}

// VCSLabelFunc gets the appropriate label for removing from version control
func VCSLabelFunc(fn *Node, label string) string {
	repo, _ := fn.Repo()
	if repo != nil {
		label = strings.Replace(label, "VCS", string(repo.Vcs()), 1)
	}
	return label
}

func (fn *Node) VCSContextMenu(m *core.Scene) {
	views.NewFuncButton(m, fn.AddToVCSSel).SetText(VCSLabelFunc(fn, "Add to VCS")).SetIcon(icons.Add).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS != vcs.Untracked, states.Disabled)
		})
	views.NewFuncButton(m, fn.DeleteFromVCSSel).SetText(VCSLabelFunc(fn, "Delete from VCS")).SetIcon(icons.Delete).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	views.NewFuncButton(m, fn.CommitToVCSSel).SetText(VCSLabelFunc(fn, "Commit to VCS")).SetIcon(icons.Star).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	views.NewFuncButton(m, fn.RevertVCSSel).SetText(VCSLabelFunc(fn, "Revert from VCS")).SetIcon(icons.Undo).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	core.NewSeparator(m)

	views.NewFuncButton(m, fn.DiffVCSSel).SetText(VCSLabelFunc(fn, "Diff VCS")).SetIcon(icons.Add).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	views.NewFuncButton(m, fn.LogVCSSel).SetText(VCSLabelFunc(fn, "Log VCS")).SetIcon(icons.List).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	views.NewFuncButton(m, fn.BlameVCSSel).SetText(VCSLabelFunc(fn, "Blame VCS")).SetIcon(icons.CreditScore).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
}

func (fn *Node) ContextMenu(m *core.Scene) {
	core.NewButton(m).SetText("Info").SetIcon(icons.Info).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		}).OnClick(func(e events.Event) {
		fn.This().(Filer).ShowFileInfo()
	})

	core.NewButton(m).SetText("Open").SetIcon(icons.Open).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		}).OnClick(func(e events.Event) {
		fn.This().(Filer).OpenFilesDefault()
	})
	core.NewSeparator(m)

	core.NewButton(m).SetText("Duplicate").SetIcon(icons.Copy).
		SetKey(keymap.Duplicate).Style(func(s *styles.Style) {
		s.SetState(!fn.HasSelection(), states.Disabled)
	}).OnClick(func(e events.Event) {
		fn.This().(Filer).DuplicateFiles()
	})
	core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).
		SetKey(keymap.Delete).Style(func(s *styles.Style) {
		s.SetState(!fn.HasSelection(), states.Disabled)
	}).OnClick(func(e events.Event) {
		fn.This().(Filer).DeleteFiles()
	})
	core.NewButton(m).SetText("Rename").SetIcon(icons.NewText).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		}).OnClick(func(e events.Event) {
		fn.This().(Filer).RenameFiles()
	})
	core.NewSeparator(m)

	views.NewFuncButton(m, fn.OpenAll).SetText("Open all").SetIcon(icons.KeyboardArrowDown).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsDir(), states.Disabled)
		})
	views.NewFuncButton(m, fn.CloseAll).SetIcon(icons.KeyboardArrowRight).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsDir(), states.Disabled)
		})
	views.NewFuncButton(m, fn.SortBys).SetText("Sort by").SetIcon(icons.Sort).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsDir(), states.Disabled)
		})
	core.NewSeparator(m)

	views.NewFuncButton(m, fn.NewFiles).SetText("New file").SetIcon(icons.OpenInNew).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	views.NewFuncButton(m, fn.NewFolders).SetText("New folder").SetIcon(icons.CreateNewFolder).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	core.NewSeparator(m)

	fn.VCSContextMenu(m)
	core.NewSeparator(m)

	views.NewFuncButton(m, fn.RemoveFromExterns).SetIcon(icons.Delete).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})

	core.NewSeparator(m)
	core.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keymap.Copy).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		}).
		OnClick(func(e events.Event) {
			fn.Copy(true)
		})
	core.NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keymap.Cut).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		}).
		OnClick(func(e events.Event) {
			fn.Cut()
		})
	pbt := core.NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keymap.Paste).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		}).
		OnClick(func(e events.Event) {
			fn.Paste()
		})
	cb := fn.Events().Clipboard()
	if cb != nil {
		pbt.SetState(cb.IsEmpty(), states.Disabled)
	}
}
