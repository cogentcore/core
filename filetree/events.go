// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"fmt"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/vci"
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
	kf := keyfun.Of(kt.KeyChord())
	selMode := events.SelectModeBits(kt.Modifiers())

	if selMode == events.SelectOne {
		if fn.SelectMode() {
			selMode = events.ExtendContinuous
		}
	}

	// first all the keys that work for ReadOnly and active
	if !fn.IsReadOnly() && !kt.IsHandled() {
		switch kf {
		case keyfun.Delete:
			fn.DeleteFiles()
			kt.SetHandled()
			// todo: remove when gi issue 237 is resolved
		case keyfun.Backspace:
			fn.DeleteFiles()
			kt.SetHandled()
		case keyfun.Duplicate:
			fn.DuplicateFiles()
			kt.SetHandled()
		case keyfun.Insert: // New File
			giv.CallFunc(fn, fn.NewFile)
			kt.SetHandled()
		case keyfun.InsertAfter: // New Folder
			giv.CallFunc(fn, fn.NewFolder)
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
	giv.NewFuncButton(m, fn.AddToVCSSel).SetText(VCSLabelFunc(fn, "Add to VCS")).SetIcon(icons.Add).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.Vcs != vci.Untracked, states.Disabled)
		})
	giv.NewFuncButton(m, fn.DeleteFromVCSSel).SetText(VCSLabelFunc(fn, "Delete from VCS")).SetIcon(icons.Delete).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.Vcs == vci.Untracked, states.Disabled)
		})
	giv.NewFuncButton(m, fn.CommitToVCSSel).SetText(VCSLabelFunc(fn, "Commit to VCS")).SetIcon(icons.Star).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.Vcs == vci.Untracked, states.Disabled)
		})
	giv.NewFuncButton(m, fn.RevertVCSSel).SetText(VCSLabelFunc(fn, "Revert from VCS")).SetIcon(icons.Undo).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.Vcs == vci.Untracked, states.Disabled)
		})
	core.NewSeparator(m)

	giv.NewFuncButton(m, fn.DiffVCSSel).SetText(VCSLabelFunc(fn, "Diff VCS")).SetIcon(icons.Add).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.Vcs == vci.Untracked, states.Disabled)
		})
	giv.NewFuncButton(m, fn.LogVCSSel).SetText(VCSLabelFunc(fn, "Log VCS")).SetIcon(icons.List).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.Vcs == vci.Untracked, states.Disabled)
		})
	giv.NewFuncButton(m, fn.BlameVCSSel).SetText(VCSLabelFunc(fn, "Blame VCS")).SetIcon(icons.CreditScore).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.Vcs == vci.Untracked, states.Disabled)
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
		SetKey(keyfun.Duplicate).Style(func(s *styles.Style) {
		s.SetState(!fn.HasSelection(), states.Disabled)
	}).OnClick(func(e events.Event) {
		fn.This().(Filer).DuplicateFiles()
	})
	core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).
		SetKey(keyfun.Delete).Style(func(s *styles.Style) {
		s.SetState(!fn.HasSelection(), states.Disabled)
	}).OnClick(func(e events.Event) {
		fn.This().(Filer).DeleteFiles()
	})
	core.NewButton(m).SetText("Rename").SetIcon(icons.NewLabel).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		}).OnClick(func(e events.Event) {
		fn.This().(Filer).RenameFiles()
	})
	core.NewSeparator(m)

	giv.NewFuncButton(m, fn.OpenAll).SetText("Open all").SetIcon(icons.KeyboardArrowDown).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsDir(), states.Disabled)
		})
	giv.NewFuncButton(m, fn.CloseAll).SetIcon(icons.KeyboardArrowRight).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsDir(), states.Disabled)
		})
	giv.NewFuncButton(m, fn.SortBys).SetText("Sort by").SetIcon(icons.Sort).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsDir(), states.Disabled)
		})
	core.NewSeparator(m)

	giv.NewFuncButton(m, fn.NewFiles).SetText("New file").SetIcon(icons.OpenInNew).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	giv.NewFuncButton(m, fn.NewFolders).SetText("New folder").SetIcon(icons.CreateNewFolder).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	core.NewSeparator(m)

	fn.VCSContextMenu(m)
	core.NewSeparator(m)

	giv.NewFuncButton(m, fn.RemoveFromExterns).SetIcon(icons.Delete).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})

	core.NewSeparator(m)
	core.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keyfun.Copy).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		}).
		OnClick(func(e events.Event) {
			fn.Copy(true)
		})
	core.NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keyfun.Cut).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		}).
		OnClick(func(e events.Event) {
			fn.Cut()
		})
	pbt := core.NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keyfun.Paste).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		}).
		OnClick(func(e events.Event) {
			fn.Paste()
		})
	cb := fn.EventMgr().Clipboard()
	if cb != nil {
		pbt.SetState(cb.IsEmpty(), states.Disabled)
	}
}
