// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gox/errors"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/vcs"
)

func (ft *Tree) OnInit() {
	ft.Node.OnInit()
	ft.FRoot = ft
	ft.FileNodeType = NodeType
	ft.OpenDepth = 4
}

func (fn *Node) OnInit() {
	fn.TreeView.OnInit()
	fn.HandleEvents()
	fn.SetStyles()
	fn.ContextMenus = nil // do not include treeview
	fn.AddContextMenu(fn.ContextMenu)
}

func (fn *Node) SetStyles() {
	fn.Style(func(s *styles.Style) {
		status := fn.Info.VCS
		s.Font.Weight = styles.WeightNormal
		s.Font.Style = styles.FontNormal
		if fn.IsExec() && !fn.IsDir() {
			s.Font.Weight = styles.WeightBold // todo: somehow not working
		}
		if fn.Buffer != nil {
			s.Font.Style = styles.Italic
		}
		switch {
		case status == vcs.Untracked:
			s.Color = errors.Must1(gradient.FromString("#808080"))
		case status == vcs.Modified:
			s.Color = errors.Must1(gradient.FromString("#4b7fd1"))
		case status == vcs.Added:
			s.Color = errors.Must1(gradient.FromString("#008800"))
		case status == vcs.Deleted:
			s.Color = errors.Must1(gradient.FromString("#ff4252"))
		case status == vcs.Conflicted:
			s.Color = errors.Must1(gradient.FromString("#ce8020"))
		case status == vcs.Updated:
			s.Color = errors.Must1(gradient.FromString("#008060"))
		case status == vcs.Stored:
			s.Color = colors.C(colors.Scheme.OnSurface)
		}
	})
	fn.OnWidgetAdded(func(w core.Widget) {
		switch w.PathFrom(fn) {
		case "parts":
			parts := w.(*core.Layout)
			w.OnClick(func(e events.Event) {
				fn.OpenEmptyDir()
			})
			parts.OnDoubleClick(func(e events.Event) {
				if fn.FRoot != nil && fn.FRoot.DoubleClickFun != nil {
					fn.FRoot.DoubleClickFun(e)
				} else {
					if fn.IsDir() && fn.OpenEmptyDir() {
						e.SetHandled()
					}
				}
			})
		case "parts/branch":
			sw := w.(*core.Switch)
			sw.Type = core.SwitchCheckbox
			sw.SetIcons(icons.FolderOpen, icons.Folder, icons.Blank)
		}
	})
}
