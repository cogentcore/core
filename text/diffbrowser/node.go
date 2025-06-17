// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diffbrowser

import (
	"log/slog"
	"os"
	"path/filepath"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// Node is an element in the diff tree
type Node struct {
	core.Tree

	// file names (full path) being compared. Name of node is just the filename.
	// Typically A is the older, base version and B is the newer one being compared.
	FileA, FileB string

	// VCS revisions for files if applicable
	RevA, RevB string

	// Status of the change from A to B: A=Added, D=Deleted, M=Modified, R=Renamed
	Status string

	// Text content of the files
	TextA, TextB string

	// Info about the B file, for getting icons etc
	Info fileinfo.FileInfo
}

func (tn *Node) Init() {
	tn.Tree.Init()
	tn.IconOpen = icons.FolderOpen
	tn.IconClosed = icons.Folder
	tn.ContextMenus = nil
	tn.AddContextMenu(tn.ContextMenu)

	tn.Styler(func(s *styles.Style) {
		s.IconSize.Set(units.Em(1))
	})
	tn.Parts.AsWidget().OnDoubleClick(func(e events.Event) {
		if tn.HasChildren() {
			return
		}
		br := tn.Browser()
		if br == nil {
			return
		}
		sels := tn.GetSelectedNodes()
		if sels != nil {
			br.ViewDiff(tn)
		}
	})
	tn.Parts.Styler(func(s *styles.Style) {
		s.Gap.X.Em(0.4)
	})
}

// Browser returns the parent browser
func (tn *Node) Browser() *Browser {
	return tree.ParentByType[*Browser](tn)
}

func (tn *Node) ContextMenu(m *core.Scene) {
	vd := core.NewButton(m).SetText("View Diffs").SetIcon(icons.Add)
	vd.Styler(func(s *styles.Style) {
		s.SetState(!tn.HasSelection(), states.Disabled)
	})
	vd.OnClick(func(e events.Event) {
		br := tn.Browser()
		if br == nil {
			return
		}
		sels := tn.GetSelectedNodes()
		sn := sels[len(sels)-1].(*Node)
		br.ViewDiff(sn)
	})
}

// DiffDirs creates a tree of files within the two paths,
// where the files have the same names, yet differ in content.
// The excludeFile function, if non-nil, will exclude files or
// directories from consideration if it returns true.
func (br *Browser) DiffDirs(pathA, pathB string, excludeFile func(fname string) bool) {
	br.PathA = pathA
	br.PathB = pathB
	tv := br.Tree()
	tv.SetText(fsx.DirAndFile(pathA))
	br.diffDirsAt(pathA, pathB, tv, excludeFile)
}

// diffDirsAt creates a tree of files with the same names
// that differ within two dirs.
func (br *Browser) diffDirsAt(pathA, pathB string, node *Node, excludeFile func(fname string) bool) {
	da := fsx.Dirs(pathA)
	db := fsx.Dirs(pathB)

	node.SetFileA(pathA).SetFileB(pathB)

	for _, pa := range da {
		if excludeFile != nil && excludeFile(pa) {
			continue
		}
		for _, pb := range db {
			if pa == pb {
				nn := NewNode(node)
				nn.SetText(pa)
				br.diffDirsAt(filepath.Join(pathA, pa), filepath.Join(pathB, pb), nn, excludeFile)
			}
		}
	}

	fsa := fsx.Filenames(pathA)
	fsb := fsx.Filenames(pathB)

	for _, fa := range fsa {
		isDir := false
		for _, pa := range da {
			if fa == pa {
				isDir = true
				break
			}
		}
		if isDir {
			continue
		}
		if excludeFile != nil && excludeFile(fa) {
			continue
		}
		for _, fb := range fsb {
			if fa != fb {
				continue
			}
			pfa := filepath.Join(pathA, fa)
			pfb := filepath.Join(pathB, fb)

			ca, err := os.ReadFile(pfa)
			if err != nil {
				slog.Error(err.Error())
				continue
			}
			cb, err := os.ReadFile(pfb)
			if err != nil {
				slog.Error(err.Error())
				continue
			}
			sa := string(ca)
			sb := string(cb)
			if sa == sb {
				continue
			}
			nn := NewNode(node)
			nn.SetText(fa)
			nn.SetFileA(pfa).SetFileB(pfb).SetTextA(sa).SetTextB(sb)
			nn.Info.InitFile(pfb)
			nn.IconLeaf = nn.Info.Ic
		}
	}
}
