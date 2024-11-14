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

// vcsLabelFunc gets the appropriate label for removing from version control
func vcsLabelFunc(fn *Node, label string) string {
	repo, _ := fn.Repo()
	if repo != nil {
		label = strings.Replace(label, "VCS", string(repo.Vcs()), 1)
	}
	return label
}

func (fn *Node) VCSContextMenu(m *core.Scene) {
	if fn.FileRoot().FS != nil {
		return
	}
	core.NewFuncButton(m).SetFunc(fn.addToVCSSelected).SetText(vcsLabelFunc(fn, "Add to VCS")).SetIcon(icons.Add).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS != vcs.Untracked, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.deleteFromVCSSelected).SetText(vcsLabelFunc(fn, "Delete from VCS")).SetIcon(icons.Delete).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.commitToVCSSelected).SetText(vcsLabelFunc(fn, "Commit to VCS")).SetIcon(icons.Star).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.revertVCSSelected).SetText(vcsLabelFunc(fn, "Revert from VCS")).SetIcon(icons.Undo).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	core.NewSeparator(m)

	core.NewFuncButton(m).SetFunc(fn.diffVCSSelected).SetText(vcsLabelFunc(fn, "Diff VCS")).SetIcon(icons.Add).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.logVCSSelected).SetText(vcsLabelFunc(fn, "Log VCS")).SetIcon(icons.List).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.blameVCSSelected).SetText(vcsLabelFunc(fn, "Blame VCS")).SetIcon(icons.CreditScore).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.VCS == vcs.Untracked, states.Disabled)
		})
}

func (fn *Node) contextMenu(m *core.Scene) {
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

	core.NewFuncButton(m).SetFunc(fn.openAll).SetText("Open all").SetIcon(icons.KeyboardArrowDown).SetEnabled(fn.HasSelection() && fn.IsDir())
	core.NewFuncButton(m).SetFunc(fn.CloseAll).SetIcon(icons.KeyboardArrowRight).SetEnabled(fn.HasSelection() && fn.IsDir())
	core.NewFuncButton(m).SetFunc(fn.sortBys).SetText("Sort by").SetIcon(icons.Sort).SetEnabled(fn.HasSelection() && fn.IsDir())
	core.NewSeparator(m)

	core.NewFuncButton(m).SetFunc(fn.newFiles).SetText("New file").SetIcon(icons.OpenInNew).SetEnabled(fn.HasSelection())
	core.NewFuncButton(m).SetFunc(fn.newFolders).SetText("New folder").SetIcon(icons.CreateNewFolder).SetEnabled(fn.HasSelection())
	core.NewSeparator(m)

	fn.VCSContextMenu(m)
	core.NewSeparator(m)

	core.NewFuncButton(m).SetFunc(fn.removeFromExterns).SetIcon(icons.Delete).SetEnabled(fn.HasSelection())

	core.NewSeparator(m)
	core.NewFuncButton(m).SetFunc(fn.Copy).SetIcon(icons.Copy).SetKey(keymap.Copy).SetEnabled(fn.HasSelection())
	core.NewFuncButton(m).SetFunc(fn.Cut).SetIcon(icons.Cut).SetKey(keymap.Cut).SetEnabled(fn.HasSelection())
	paste := core.NewFuncButton(m).SetFunc(fn.Paste).SetIcon(icons.Paste).SetKey(keymap.Paste).SetEnabled(fn.HasSelection())
	cb := fn.Events().Clipboard()
	if cb != nil {
		paste.SetState(cb.IsEmpty(), states.Disabled)
	}
}
