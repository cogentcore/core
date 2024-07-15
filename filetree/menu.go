// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"strings"

	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/system"
)

// VCSLabelFunc gets the appropriate label for removing from version control
func VCSLabelFunc(fn *Node, label string) string {
	repo, _ := fn.Repo()
	if repo != nil {
		label = strings.Replace(label, "VCS", string(repo.Vcs()), 1)
	}
	return label
}

func (fn *Node) VCSContextMenu(m *core.Scene) {
	core.NewFuncButton(m).SetFunc(fn.AddToVCSSel).SetText(VCSLabelFunc(fn, "Add to VCS")).SetIcon(icons.Add).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS != vcs.Untracked, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.DeleteFromVCSSel).SetText(VCSLabelFunc(fn, "Delete from VCS")).SetIcon(icons.Delete).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.CommitToVCSSel).SetText(VCSLabelFunc(fn, "Commit to VCS")).SetIcon(icons.Star).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.RevertVCSSel).SetText(VCSLabelFunc(fn, "Revert from VCS")).SetIcon(icons.Undo).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	core.NewSeparator(m)

	core.NewFuncButton(m).SetFunc(fn.DiffVCSSel).SetText(VCSLabelFunc(fn, "Diff VCS")).SetIcon(icons.Add).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.LogVCSSel).SetText(VCSLabelFunc(fn, "Log VCS")).SetIcon(icons.List).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.BlameVCSSel).SetText(VCSLabelFunc(fn, "Blame VCS")).SetIcon(icons.CreditScore).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
}

func (fn *Node) ContextMenu(m *core.Scene) {
	core.NewFuncButton(m).SetFunc(fn.showFileInfo).SetText("Info").SetIcon(icons.Info).SetEnabled(fn.HasSelection())
	open := core.NewFuncButton(m).SetFunc(fn.OpenFilesDefault).SetText("Open").SetIcon(icons.Open)
	open.SetEnabled(fn.HasSelection())
	if core.TheApp.Platform() == system.Web {
		open.SetText("Download").SetIcon(icons.Download).SetTooltip("Download this file to your device")
	}
	core.NewSeparator(m)

	core.NewFuncButton(m).SetFunc(fn.duplicateFiles).SetText("Duplicate").SetIcon(icons.Copy).SetKey(keymap.Duplicate).SetEnabled(fn.HasSelection())
	core.NewFuncButton(m).SetFunc(fn.deleteFiles).SetText("Delete").SetIcon(icons.Delete).SetKey(keymap.Delete).SetEnabled(fn.HasSelection())
	core.NewFuncButton(m).SetFunc(fn.This.(Filer).RenameFiles).SetText("Rename").SetIcon(icons.NewLabel).SetEnabled(fn.HasSelection())
	core.NewSeparator(m)

	core.NewFuncButton(m).SetFunc(fn.OpenAll).SetText("Open all").SetIcon(icons.KeyboardArrowDown).SetEnabled(fn.HasSelection() && fn.IsDir())
	core.NewFuncButton(m).SetFunc(fn.CloseAll).SetIcon(icons.KeyboardArrowRight).SetEnabled(fn.HasSelection() && fn.IsDir())
	core.NewFuncButton(m).SetFunc(fn.SortBys).SetText("Sort by").SetIcon(icons.Sort).SetEnabled(fn.HasSelection() && fn.IsDir())
	core.NewSeparator(m)

	core.NewFuncButton(m).SetFunc(fn.newFiles).SetText("New file").SetIcon(icons.OpenInNew).SetEnabled(fn.HasSelection())
	core.NewFuncButton(m).SetFunc(fn.newFolders).SetText("New folder").SetIcon(icons.CreateNewFolder).SetEnabled(fn.HasSelection())
	core.NewSeparator(m)

	fn.VCSContextMenu(m)
	core.NewSeparator(m)

	core.NewFuncButton(m).SetFunc(fn.RemoveFromExterns).SetIcon(icons.Delete).SetEnabled(fn.HasSelection())

	core.NewSeparator(m)
	core.NewFuncButton(m).SetFunc(fn.Copy).SetIcon(icons.Copy).SetKey(keymap.Copy).SetEnabled(fn.HasSelection())
	core.NewFuncButton(m).SetFunc(fn.Cut).SetIcon(icons.Cut).SetKey(keymap.Cut).SetEnabled(fn.HasSelection())
	paste := core.NewFuncButton(m).SetFunc(fn.Paste).SetIcon(icons.Paste).SetKey(keymap.Paste).SetEnabled(fn.HasSelection())
	cb := fn.Events().Clipboard()
	if cb != nil {
		paste.SetState(cb.IsEmpty(), states.Disabled)
	}
}
