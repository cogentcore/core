// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/textbuf"
	"cogentcore.org/core/tree"
)

// FirstVCS returns the first VCS repository starting from this node and going down.
// also returns the node having that repository
func (fn *Node) FirstVCS() (vcs.Repo, *Node) {
	var repo vcs.Repo
	var rnode *Node
	fn.WidgetWalkDown(func(wi core.Widget, wb *core.WidgetBase) bool {
		sfn := AsNode(wi)
		if sfn == nil {
			return tree.Continue
		}
		if sfn.DirRepo != nil {
			repo = sfn.DirRepo
			rnode = sfn
			return tree.Break
		}
		return tree.Continue
	})
	return repo, rnode
}

// detectVCSRepo detects and configures DirRepo if this directory is root of
// a VCS repository.  if updateFiles is true, gets the files in the dir.
// returns true if a repository was newly found here.
func (fn *Node) detectVCSRepo(updateFiles bool) bool {
	repo, _ := fn.Repo()
	if repo != nil {
		return false
	}
	path := string(fn.Filepath)
	rtyp := vcs.DetectRepo(path)
	if rtyp == "" {
		return false
	}
	var err error
	repo, err = vcs.NewRepo("origin", path)
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	fn.DirRepo = repo
	if updateFiles {
		fn.updateRepoFiles()
	}
	return true
}

// Repo returns the version control repository associated with this file,
// and the node for the directory where the repo is based.
// Goes up the tree until a repository is found.
func (fn *Node) Repo() (vcs.Repo, *Node) {
	if fn.isExternal() {
		return nil, nil
	}
	if fn.DirRepo != nil {
		return fn.DirRepo, fn
	}
	var repo vcs.Repo
	var rnode *Node
	fn.WalkUpParent(func(k tree.Node) bool {
		sfn := AsNode(k)
		if sfn == nil {
			return tree.Break
		}
		if sfn.IsIrregular() {
			return tree.Break
		}
		if sfn.DirRepo != nil {
			repo = sfn.DirRepo
			rnode = sfn
			return tree.Break
		}
		return tree.Continue
	})
	return repo, rnode
}

func (fn *Node) updateRepoFiles() {
	if fn.DirRepo == nil {
		return
	}
	fn.repoFiles, _ = fn.DirRepo.Files()
}

// addToVCSSelected adds selected files to version control system
func (fn *Node) addToVCSSelected() { //types:add
	fn.SelectedFunc(func(sn *Node) {
		sn.AddToVCS()
	})
}

// AddToVCS adds file to version control
func (fn *Node) AddToVCS() {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	// fmt.Printf("adding to vcs: %v\n", fn.FPath)
	err := repo.Add(string(fn.Filepath))
	if errors.Log(err) == nil {
		fn.Info.VCS = vcs.Added
		fn.NeedsRender()
	}
}

// deleteFromVCSSelected removes selected files from version control system
func (fn *Node) deleteFromVCSSelected() { //types:add
	fn.SelectedFunc(func(sn *Node) {
		sn.deleteFromVCS()
	})
}

// deleteFromVCS removes file from version control
func (fn *Node) deleteFromVCS() {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	// fmt.Printf("deleting remote from vcs: %v\n", fn.FPath)
	err := repo.DeleteRemote(string(fn.Filepath))
	if fn != nil && errors.Log(err) == nil {
		fn.Info.VCS = vcs.Deleted
		fn.NeedsRender()
	}
}

// commitToVCSSelected commits to version control system based on last selected file
func (fn *Node) commitToVCSSelected() { //types:add
	done := false
	fn.SelectedFunc(func(sn *Node) {
		if !done {
			core.CallFunc(sn, fn.commitToVCS)
			done = true
		}
	})
}

// commitToVCS commits file changes to version control system
func (fn *Node) commitToVCS(message string) (err error) {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	if fn.Info.VCS == vcs.Untracked {
		return errors.New("file not in vcs repo: " + string(fn.Filepath))
	}
	err = repo.CommitFile(string(fn.Filepath), message)
	if err != nil {
		return err
	}
	fn.Info.VCS = vcs.Stored
	fn.NeedsRender()
	return err
}

// revertVCSSelected removes selected files from version control system
func (fn *Node) revertVCSSelected() { //types:add
	fn.SelectedFunc(func(sn *Node) {
		sn.revertVCS()
	})
}

// revertVCS reverts file changes since last commit
func (fn *Node) revertVCS() (err error) {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	if fn.Info.VCS == vcs.Untracked {
		return errors.New("file not in vcs repo: " + string(fn.Filepath))
	}
	err = repo.RevertFile(string(fn.Filepath))
	if err != nil {
		return err
	}
	if fn.Info.VCS == vcs.Modified {
		fn.Info.VCS = vcs.Stored
	} else if fn.Info.VCS == vcs.Added {
		// do nothing - leave in "added" state
	}
	if fn.Buffer != nil {
		fn.Buffer.Revert()
	}
	fn.NeedsRender()
	return err
}

// diffVCSSelected shows the diffs between two versions of selected files, given by the
// revision specifiers -- if empty, defaults to A = current HEAD, B = current WC file.
// -1, -2 etc also work as universal ways of specifying prior revisions.
// Diffs are shown in a DiffEditorDialog.
func (fn *Node) diffVCSSelected(rev_a string, rev_b string) { //types:add
	fn.SelectedFunc(func(sn *Node) {
		sn.diffVCS(rev_a, rev_b)
	})
}

// diffVCS shows the diffs between two versions of this file, given by the
// revision specifiers -- if empty, defaults to A = current HEAD, B = current WC file.
// -1, -2 etc also work as universal ways of specifying prior revisions.
// Diffs are shown in a DiffEditorDialog.
func (fn *Node) diffVCS(rev_a, rev_b string) error {
	repo, _ := fn.Repo()
	if repo == nil {
		return errors.New("file not in vcs repo: " + string(fn.Filepath))
	}
	if fn.Info.VCS == vcs.Untracked {
		return errors.New("file not in vcs repo: " + string(fn.Filepath))
	}
	_, err := texteditor.DiffEditorDialogFromRevs(fn, repo, string(fn.Filepath), fn.Buffer, rev_a, rev_b)
	return err
}

// logVCSSelected shows the VCS log of commits for selected files.
func (fn *Node) logVCSSelected() { //types:add
	fn.SelectedFunc(func(sn *Node) {
		sn.LogVCS(false, "")
	})
}

// LogVCS shows the VCS log of commits for this file, optionally with a
// since date qualifier: If since is non-empty, it should be
// a date-like expression that the VCS will understand, such as
// 1/1/2020, yesterday, last year, etc.  SVN only understands a
// number as a maximum number of items to return.
// If allFiles is true, then the log will show revisions for all files, not just
// this one.
// Returns the Log and also shows it in a VCSLog which supports further actions.
func (fn *Node) LogVCS(allFiles bool, since string) (vcs.Log, error) {
	repo, _ := fn.Repo()
	if repo == nil {
		return nil, errors.New("file not in vcs repo: " + string(fn.Filepath))
	}
	if fn.Info.VCS == vcs.Untracked {
		return nil, errors.New("file not in vcs repo: " + string(fn.Filepath))
	}
	fnm := string(fn.Filepath)
	if allFiles {
		fnm = ""
	}
	lg, err := repo.Log(fnm, since)
	if err != nil {
		return lg, err
	}
	vcsLogDialog(nil, repo, lg, fnm, since)
	return lg, nil
}

// blameVCSSelected shows the VCS blame report for this file, reporting for each line
// the revision and author of the last change.
func (fn *Node) blameVCSSelected() { //types:add
	fn.SelectedFunc(func(sn *Node) {
		sn.blameVCS()
	})
}

// blameDialog opens a dialog for displaying VCS blame data using textview.TwinViews.
// blame is the annotated blame code, while fbytes is the original file contents.
func blameDialog(ctx core.Widget, fname string, blame, fbytes []byte) *texteditor.TwinEditors {
	title := "VCS Blame: " + fsx.DirAndFile(fname)

	d := core.NewBody().AddTitle(title)
	tv := texteditor.NewTwinEditors(d)
	tv.SetSplits(.3, .7)
	tv.SetFiles(fname, fname)
	flns := bytes.Split(fbytes, []byte("\n"))
	lns := bytes.Split(blame, []byte("\n"))
	nln := min(len(lns), len(flns))
	blns := make([][]byte, nln)
	stidx := 0
	for i := 0; i < nln; i++ {
		fln := flns[i]
		bln := lns[i]
		if stidx == 0 {
			if len(fln) == 0 {
				stidx = len(bln)
			} else {
				stidx = bytes.LastIndex(bln, fln)
			}
		}
		blns[i] = bln[:stidx]
	}
	btxt := bytes.Join(blns, []byte("\n")) // makes a copy, so blame is disposable now
	tv.BufferA.SetText(btxt)
	tv.BufferB.SetText(fbytes)
	tv.Update()

	tva, tvb := tv.Editors()
	tva.Styler(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePre
		s.Min.X.Ch(30)
		s.Min.Y.Em(40)
	})
	tvb.Styler(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePre
		s.Min.X.Ch(80)
		s.Min.Y.Em(40)
	})
	d.AddOKOnly()
	d.RunWindowDialog(ctx)
	return tv
}

// blameVCS shows the VCS blame report for this file, reporting for each line
// the revision and author of the last change.
func (fn *Node) blameVCS() ([]byte, error) {
	repo, _ := fn.Repo()
	if repo == nil {
		return nil, errors.New("file not in vcs repo: " + string(fn.Filepath))
	}
	if fn.Info.VCS == vcs.Untracked {
		return nil, errors.New("file not in vcs repo: " + string(fn.Filepath))
	}
	fnm := string(fn.Filepath)
	fb, err := textbuf.FileBytes(fnm)
	if err != nil {
		return nil, err
	}
	blm, err := repo.Blame(fnm)
	if err != nil {
		return blm, err
	}
	blameDialog(nil, fnm, blm, fb)
	return blm, nil
}

// UpdateAllVCS does an update on any repositories below this one in file tree
func (fn *Node) UpdateAllVCS() {
	fn.WidgetWalkDown(func(wi core.Widget, wb *core.WidgetBase) bool {
		sfn := AsNode(wi)
		if sfn == nil {
			return tree.Continue
		}
		if !sfn.IsDir() {
			return tree.Continue
		}
		if sfn.DirRepo == nil {
			if !sfn.detectVCSRepo(false) {
				return tree.Continue
			}
		}
		repo := sfn.DirRepo
		fmt.Printf("Updating %v repository: %s from: %s\n", repo.Vcs(), sfn.RelativePath(), repo.Remote())
		err := repo.Update()
		if err != nil {
			fmt.Printf("error: %v\n", err)
		}
		return tree.Break
	})
}

// versionControlSystems is a list of supported Version Control Systems.
// These must match the VCS Types from vcs which in turn
// is based on masterminds/vcs
var versionControlSystems = []string{"git", "svn", "bzr", "hg"}

// IsVersionControlSystem returns true if the given string matches one of the
// standard VersionControlSystems -- uses lowercase version of str.
func IsVersionControlSystem(str string) bool {
	stl := strings.ToLower(str)
	for _, vcn := range versionControlSystems {
		if stl == vcn {
			return true
		}
	}
	return false
}

// VersionControlName is the name of a version control system
type VersionControlName string

// Value registers [core.Chooser] as the [core.Value] widget
// for [VersionControlName]
func (kn VersionControlName) Value() core.Value {
	return core.NewChooser().SetStrings(versionControlSystems...)
}
