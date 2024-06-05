// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diffbrowser

import (
	"log/slog"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/stringsx"
	"cogentcore.org/core/base/vcs"
)

// NewDiffBrowserVCS returns a new diff browser for files that differ
// between two given revisions in the repository.
func NewDiffBrowserVCS(repo vcs.Repo, revA, revB string) {
	brow, b := NewBrowserWindow()
	brow.DiffVCS(repo, revA, revB)
	b.RunWindow()
}

// DiffVCS creates a tree of files changed in given revision.
func (br *Browser) DiffVCS(repo vcs.Repo, revA, revB string) {
	cinfo, err := repo.FilesChanged(revA, revB, false)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	br.PathA = repo.LocalPath()
	br.PathB = br.PathA
	files := stringsx.SplitLines(string(cinfo))
	tv := br.Tree()
	tv.SetText(dirs.DirAndFile(br.PathA))
	cdir := ""
	var cdirs []string
	var cnodes []*Node
	root := br.Tree()
	for _, fl := range files {
		fd := strings.Fields(fl)
		if len(fd) < 2 {
			continue
		}
		fpa := strings.TrimSpace(fd[1])
		dir, fn := filepath.Split(fpa)
		dir = filepath.Dir(dir)
		if dir != cdir {
			dirs := strings.Split(dir, "/")
			nd := len(dirs)
			mn := min(len(cdirs), nd)
			di := 0
			for i := 0; i < mn; i++ {
				if cdirs[i] != dirs[i] {
					break
				} else {
					di = i
				}
			}
			cnodes = cnodes[:di]
			for i := di; i < nd; i++ {
				var nn *Node
				if i == 0 {
					nn = NewNode(root)
				} else {
					nn = NewNode(cnodes[i-1])
				}
				dp := filepath.Join(br.PathA, filepath.Join(dirs[:i+1]...))
				nn.SetFileA(dp).SetFileB(dp)
				nn.SetText(dirs[i])
				cnodes = append(cnodes, nn)
			}
			cdir = dir
			cdirs = dirs
		}
		var nn *Node
		nd := len(cnodes)
		if nd == 0 {
			nn = NewNode(root)
		} else {
			nn = NewNode(cnodes[nd-1])
		}
		dp := filepath.Join(br.PathA, filepath.Join(cdir), fn)
		nn.SetFileA(dp).SetFileB(dp).SetRevA(revA).SetRevB(revB)
		nn.SetText(fn)
		fbA, err := repo.FileContents(dp, revA)
		if err != nil {
			slog.Error(err.Error())
		}
		fbB, err := repo.FileContents(dp, revB)
		if err != nil {
			slog.Error(err.Error())
		}
		nn.SetTextA(string(fbA)).SetTextB(string(fbB))
	}
}
