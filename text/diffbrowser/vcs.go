// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diffbrowser

import (
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/fsx"
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

// DiffVCS creates a tree of the files changed between the given revisions
// (see [vcs.Repo.FilesChanged]), with a node for each directory, which has
// its subdirectories and then its files, each sorted by name, as in a file
// tree.
func (br *Browser) DiffVCS(repo vcs.Repo, revA, revB string) {
	cinfo, err := repo.FilesChanged(revA, revB, false)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	br.PathA = repo.LocalPath()
	br.PathB = br.PathA
	root := br.Tree()
	root.SetText(fsx.DirAndFile(br.PathA))
	top := parseChanges(stringsx.SplitLines(string(cinfo)))
	br.makeChangeNodes(root, top, "", repo, revA, revB)
}

// fileChange is a file changed between two revisions.
type fileChange struct {
	// status is the status of the change: A, D, M, or R (renamed).
	status string

	// pathA and pathB are the paths of the file in the two revisions,
	// relative to the repository, which differ if it was renamed.
	pathA, pathB string
}

// changeDir is a directory of the changed files, with its subdirectories.
type changeDir struct {
	dirs  map[string]*changeDir
	files []fileChange
}

// parseChanges returns the top directory of the changed files of the
// given lines of git diff --name-status (or the equivalent of another
// VCS): a status, then the path, and the new path of a renamed file,
// separated by tabs (paths can have spaces), or else by spaces. Files
// are in the directories of their paths in revision B.
func parseChanges(lines []string) *changeDir {
	top := &changeDir{}
	for _, ln := range lines {
		fd := strings.Split(ln, "\t")
		if len(fd) < 2 {
			fd = strings.Fields(ln)
		}
		if len(fd) < 2 || fd[0] == "" {
			continue
		}
		fc := fileChange{status: fd[0][:1], pathA: fd[1], pathB: fd[1]}
		if len(fd) >= 3 {
			fc.pathB = fd[2]
		}
		d := top
		parts := strings.Split(filepath.ToSlash(fc.pathB), "/")
		for _, p := range parts[:len(parts)-1] {
			if d.dirs == nil {
				d.dirs = map[string]*changeDir{}
			}
			sub := d.dirs[p]
			if sub == nil {
				sub = &changeDir{}
				d.dirs[p] = sub
			}
			d = sub
		}
		d.files = append(d.files, fc)
	}
	return top
}

// makeChangeNodes makes the nodes of the given directory of changed files,
// at the given path relative to the repository, in the given parent node:
// those of its subdirectories, and then those of its files, each sorted
// by name, with the contents of the files at the revisions.
func (br *Browser) makeChangeNodes(parent *Node, d *changeDir, path string, repo vcs.Repo, revA, revB string) {
	for _, name := range slices.Sorted(maps.Keys(d.dirs)) {
		dp := filepath.Join(br.PathA, path, name)
		nn := NewNode(parent)
		nn.SetFileA(dp).SetFileB(dp)
		nn.SetText(name)
		br.makeChangeNodes(nn, d.dirs[name], filepath.Join(path, name), repo, revA, revB)
	}
	slices.SortFunc(d.files, func(a, b fileChange) int {
		return strings.Compare(filepath.Base(a.pathB), filepath.Base(b.pathB))
	})
	for _, fc := range d.files {
		dpa := filepath.Join(br.PathA, fc.pathA)
		dpb := filepath.Join(br.PathA, fc.pathB)
		nn := NewNode(parent)
		nn.SetFileA(dpa).SetFileB(dpb).SetRevA(revA).SetRevB(revB).SetStatus(fc.status)
		nn.SetText(filepath.Base(fc.pathB) + " [" + fc.status + "]")
		if fc.status != "D" {
			fbB, err := repo.FileContents(dpb, revB)
			if err != nil {
				slog.Error(err.Error())
			}
			nn.SetTextB(string(fbB))
			nn.Info.InitFile(dpb)
			nn.IconLeaf = nn.Info.Ic
		}
		if fc.status != "A" {
			fbA, err := repo.FileContents(dpa, revA)
			if err != nil {
				slog.Error(err.Error())
			}
			nn.SetTextA(string(fbA))
			if fc.status == "D" {
				nn.Info.InitFile(dpa)
				nn.IconLeaf = nn.Info.Ic
			}
		}
	}
}
