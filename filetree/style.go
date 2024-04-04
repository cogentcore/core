// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/vci"
)

func (ft *Tree) OnInit() {
	ft.Node.OnInit()
	ft.FRoot = ft
	ft.NodeType = NodeType
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
		vcs := fn.Info.Vcs
		s.Font.Weight = styles.WeightNormal
		s.Font.Style = styles.FontNormal
		if fn.IsExec() && !fn.IsDir() {
			s.Font.Weight = styles.WeightBold // todo: somehow not working
		}
		if fn.Buffer != nil {
			s.Font.Style = styles.Italic
		}
		switch {
		case vcs == vci.Untracked:
			s.Color = grr.Must1(gradient.FromString("#808080"))
		case vcs == vci.Modified:
			s.Color = grr.Must1(gradient.FromString("#4b7fd1"))
		case vcs == vci.Added:
			s.Color = grr.Must1(gradient.FromString("#008800"))
		case vcs == vci.Deleted:
			s.Color = grr.Must1(gradient.FromString("#ff4252"))
		case vcs == vci.Conflicted:
			s.Color = grr.Must1(gradient.FromString("#ce8020"))
		case vcs == vci.Updated:
			s.Color = grr.Must1(gradient.FromString("#008060"))
		case vcs == vci.Stored:
			s.Color = colors.C(colors.Scheme.OnSurface)
		}
	})
	fn.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(fn) {
		case "parts":
			parts := w.(*gi.Layout)
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
			sw := w.(*gi.Switch)
			sw.Type = gi.SwitchCheckbox
			sw.SetIcons(icons.FolderOpen, icons.Folder, icons.Blank)
		}
	})
}
