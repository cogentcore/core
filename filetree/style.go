// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/vci/v2"
)

func (fn *Node) OnInit() {
	fn.OpenDepth = 4
	// fn.Indent.SetEm(1)

	fn.HandleFileNodeEvents()
	fn.FileNodeStyles()
}

func (fn *Node) FileNodeStyles() {
	fn.TreeViewStyles()
	fn.Style(func(s *styles.Style) {
		vcs := fn.Info.Vcs
		switch {
		case fn.IsExec():
			s.Font.Weight = styles.WeightBold
		case fn.Buf != nil:
			s.Font.Style = styles.FontItalic
		case vcs == vci.Untracked:
			s.Color = grr.Must(colors.FromHex("#808080"))
		case vcs == vci.Modified:
			s.Color = grr.Must(colors.FromHex("#4b7fd1"))
		case vcs == vci.Added:
			s.Color = grr.Must(colors.FromHex("#008800"))
		case vcs == vci.Deleted:
			s.Color = grr.Must(colors.FromHex("#ff4252"))
		case vcs == vci.Conflicted:
			s.Color = grr.Must(colors.FromHex("#ce8020"))
		case vcs == vci.Updated:
			s.Color = grr.Must(colors.FromHex("#008060"))
		case vcs == vci.Stored:
			// s.Color = std
		}
	})
	fn.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(fn) {
		case "parts/branch":
			sw := w.(*gi.Switch)
			sw.Type = gi.SwitchCheckbox
			sw.IconOn = icons.FolderOpen
			sw.IconOff = icons.Folder
			sw.IconDisab = icons.Blank
			// sw.Style(func(s *styles.Style) {
			// 	s.MaxWidth.SetEm(1.5)
			// 	s.MaxHeight.SetEm(1.5)
			// })
		}
	})
}
