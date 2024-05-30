// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diffbrowser

import (
	"log/slog"
	"os"
	"path/filepath"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/tree"
)

// Node an element in the diff tree
type Node struct {
	tree.NodeBase

	// file names being compared
	FileA, FileB string

	// revisions for files, if applicable
	RevA, RevB string

	// Contents of the files
	ContentsA, ContentsB string
}

// DiffDirs creates a tree of files within the two paths,
// where the files have the same names, yet differ in content.
// The excludeFile function, if non-nil, will exclude files or
// directories from consideration if it returns true.
func (br *Browser) DiffDirs(pathA, pathB string, excludeFile func(fname string) bool) {
	br.PathA = pathA
	br.PathB = pathB
	br.Files = NewNode()
	br.Files.SetName(pathA)
	br.diffDirsAt(pathA, pathB, br.Files, excludeFile)
}

// diffDirsAt creates a tree of files with the same names
// that differ within two dirs.
func (br *Browser) diffDirsAt(pathA, pathB string, node *Node, excludeFile func(fname string) bool) {
	da := dirs.Dirs(pathA)
	db := dirs.Dirs(pathB)

	node.SetFileA(pathA).SetFileB(pathB)

	for _, pa := range da {
		if excludeFile != nil && excludeFile(pa) {
			continue
		}
		for _, pb := range db {
			if pa == pb {
				nn := NewNode(node)
				nn.SetName(pa)
				br.diffDirsAt(filepath.Join(pathA, pa), filepath.Join(pathB, pb), nn, excludeFile)
			}
		}
	}

	fsa := dirs.ExtFilenames(pathA)
	fsb := dirs.ExtFilenames(pathB)

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
			nn.SetName(fa)
			nn.SetFileA(pfa).SetFileB(pfb).SetContentsA(sa).SetContentsB(sb)
		}
	}
}
