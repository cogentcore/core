// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Masterminds/vcs"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/girl/styles"
	"goki.dev/glop/dirs"
	"goki.dev/ki/v2"
	"goki.dev/vci/v2"
)

// FirstVCS returns the first VCS repository starting from this node and going down.
// also returns the node having that repository
func (fn *Node) FirstVCS() (vci.Repo, *Node) {
	var repo vci.Repo
	var rnode *Node
	fn.WalkPre(func(k ki.Ki) bool {
		sfn := AsNode(k)
		if sfn.DirRepo != nil {
			repo = sfn.DirRepo
			rnode = sfn
			return ki.Break
		}
		return ki.Continue
	})
	return repo, rnode
}

// DetectVcsRepo detects and configures DirRepo if this directory is root of
// a VCS repository.  if updateFiles is true, gets the files in the dir.
// returns true if a repository was newly found here.
func (fn *Node) DetectVcsRepo(updateFiles bool) bool {
	repo, _ := fn.Repo()
	if repo != nil {
		if updateFiles {
			fn.UpdateRepoFiles()
		}
		return false
	}
	path := string(fn.FPath)
	rtyp := vci.DetectRepo(path)
	if rtyp == vcs.NoVCS {
		return false
	}
	var err error
	repo, err = vci.NewRepo("origin", path)
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	fn.DirRepo = repo
	if updateFiles {
		fn.UpdateRepoFiles()
	}
	return true
}

// Repo returns the version control repository associated with this file,
// and the node for the directory where the repo is based.
// Goes up the tree until a repository is found.
func (fn *Node) Repo() (vci.Repo, *Node) {
	if fn.IsExternal() {
		return nil, nil
	}
	if fn.DirRepo != nil {
		return fn.DirRepo, fn
	}
	var repo vci.Repo
	var rnode *Node
	fn.WalkUpParent(func(k ki.Ki) bool {
		if k == nil || k.This() == nil {
			return ki.Break
		}
		sfn := AsNode(k)
		if sfn == nil {
			return ki.Break
		}
		if sfn.IsIrregular() {
			return ki.Break
		}
		if sfn.DirRepo != nil {
			repo = sfn.DirRepo
			rnode = sfn
			return ki.Break
		}
		return ki.Continue
	})
	return repo, rnode
}

func (fn *Node) UpdateRepoFiles() {
	if fn.DirRepo == nil {
		return
	}
	fn.RepoFiles, _ = fn.DirRepo.Files()
}

// AddToVcsSel adds selected files to version control system
func (fn *Node) AddToVcsSel() {
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.AddToVcs()
	}
}

// AddToVcs adds file to version control
func (fn *Node) AddToVcs() {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	// fmt.Printf("adding to vcs: %v\n", fn.FPath)
	err := repo.Add(string(fn.FPath))
	if err == nil {
		fn.Info.Vcs = vci.Added
		fn.SetNeedsRender()
		return
	}
	fmt.Println(err)
}

// DeleteFromVcsSel removes selected files from version control system
func (fn *Node) DeleteFromVcsSel() {
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.DeleteFromVcs()
	}
}

// DeleteFromVcs removes file from version control
func (fn *Node) DeleteFromVcs() {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	// fmt.Printf("deleting remote from vcs: %v\n", fn.FPath)
	err := repo.DeleteRemote(string(fn.FPath))
	if fn != nil && err == nil {
		fn.Info.Vcs = vci.Deleted
		fn.SetNeedsRender()
		return
	}
	fmt.Println(err)
}

// CommitToVcsSel commits to version control system based on last selected file
func (fn *Node) CommitToVcsSel() {
	sels := fn.SelectedViews()
	n := len(sels)
	if n == 0 { // shouldn't happen
		return
	}
	sn := AsNode(sels[n-1])
	giv.NewFuncButton(sn, fn.CommitToVcs).CallFunc()
}

// CommitToVcs commits file changes to version control system
func (fn *Node) CommitToVcs(message string) (err error) {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	if fn.Info.Vcs == vci.Untracked {
		return errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	err = repo.CommitFile(string(fn.FPath), message)
	if err != nil {
		return err
	}
	fn.Info.Vcs = vci.Stored
	fn.SetNeedsRender()
	return err
}

// RevertVcsSel removes selected files from version control system
func (fn *Node) RevertVcsSel() {
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.RevertVcs()
	}
}

// RevertVcs reverts file changes since last commit
func (fn *Node) RevertVcs() (err error) {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	if fn.Info.Vcs == vci.Untracked {
		return errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	err = repo.RevertFile(string(fn.FPath))
	if err != nil {
		return err
	}
	if fn.Info.Vcs == vci.Modified {
		fn.Info.Vcs = vci.Stored
	} else if fn.Info.Vcs == vci.Added {
		// do nothing - leave in "added" state
	}
	if fn.Buf != nil {
		fn.Buf.Revert()
	}
	fn.SetNeedsRender()
	return err
}

// DiffVcsSel shows the diffs between two versions of selected files, given by the
// revision specifiers -- if empty, defaults to A = current HEAD, B = current WC file.
// -1, -2 etc also work as universal ways of specifying prior revisions.
// Diffs are shown in a DiffViewDialog.
func (fn *Node) DiffVcsSel(rev_a, rev_b string) {
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.DiffVcs(rev_a, rev_b)
	}
}

// DiffVcs shows the diffs between two versions of this file, given by the
// revision specifiers -- if empty, defaults to A = current HEAD, B = current WC file.
// -1, -2 etc also work as universal ways of specifying prior revisions.
// Diffs are shown in a DiffViewDialog.
func (fn *Node) DiffVcs(rev_a, rev_b string) error {
	repo, _ := fn.Repo()
	if repo == nil {
		return errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	if fn.Info.Vcs == vci.Untracked {
		return errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	// _, err := DiffViewDialogFromRevs(nil, repo, string(fn.FPath), fn.Buf, rev_a, rev_b)
	// return err
	return nil
}

// LogVcsSel shows the VCS log of commits for selected files, optionally with a
// since date qualifier: If since is non-empty, it should be
// a date-like expression that the VCS will understand, such as
// 1/1/2020, yesterday, last year, etc.  SVN only understands a
// number as a maximum number of items to return.
// If allFiles is true, then the log will show revisions for all files, not just
// this one.
// Returns the Log and also shows it in a VCSLogView which supports further actions.
func (fn *Node) LogVcsSel(allFiles bool, since string) {
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.LogVcs(allFiles, since)
	}
}

// LogVcs shows the VCS log of commits for this file, optionally with a
// since date qualifier: If since is non-empty, it should be
// a date-like expression that the VCS will understand, such as
// 1/1/2020, yesterday, last year, etc.  SVN only understands a
// number as a maximum number of items to return.
// If allFiles is true, then the log will show revisions for all files, not just
// this one.
// Returns the Log and also shows it in a VCSLogView which supports further actions.
func (fn *Node) LogVcs(allFiles bool, since string) (vci.Log, error) {
	repo, _ := fn.Repo()
	if repo == nil {
		return nil, errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	if fn.Info.Vcs == vci.Untracked {
		return nil, errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	fnm := string(fn.FPath)
	if allFiles {
		fnm = ""
	}
	lg, err := repo.Log(fnm, since)
	if err != nil {
		return lg, err
	}
	giv.VCSLogViewDialog(nil, repo, lg, fnm, since)
	return lg, nil
}

// BlameVcsSel shows the VCS blame report for this file, reporting for each line
// the revision and author of the last change.
func (fn *Node) BlameVcsSel() {
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.BlameVcs()
	}
}

// BlameDialog opens a dialog for displaying VCS blame data using textview.TwinViews.
// blame is the annotated blame code, while fbytes is the original file contents.
func BlameDialog(ctx gi.Widget, fname string, blame, fbytes []byte) *texteditor.TwinEditors {
	title := "VCS Blame: " + dirs.DirAndFile(fname)
	d := gi.NewDialog(ctx).Title(title).Ok()

	tv := texteditor.NewTwinEditors(d, "twin-view")
	tv.SetStretchMax()
	tv.SetFiles(fname, fname, true)
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
	tv.BufA.SetText(btxt)
	tv.BufB.SetText(fbytes)
	tv.ConfigTexts()
	tv.SetSplits(.2, .8)

	tva, tvb := tv.Editors()
	tva.Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePre
		s.Width.Ch(30)
		s.Height.Em(40)
	})
	tvb.Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePre
		s.Width.Ch(80)
		s.Height.Em(40)
	})

	// dlg.UpdateEndNoSig(true) // going to be shown
	// dlg.Open(0, 0, avp, nil)
	d.Run()
	return tv
}

// BlameVcs shows the VCS blame report for this file, reporting for each line
// the revision and author of the last change.
func (fn *Node) BlameVcs() ([]byte, error) {
	repo, _ := fn.Repo()
	if repo == nil {
		return nil, errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	if fn.Info.Vcs == vci.Untracked {
		return nil, errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	fnm := string(fn.FPath)
	fb, err := textbuf.FileBytes(fnm)
	if err != nil {
		return nil, err
	}
	blm, err := repo.Blame(fnm)
	if err != nil {
		return blm, err
	}
	BlameDialog(nil, fnm, blm, fb)
	return blm, nil
}

// UpdateAllVcs does an update on any repositories below this one in file tree
func (fn *Node) UpdateAllVcs() {
	fn.WalkPre(func(k ki.Ki) bool {
		sfn := AsNode(k)
		if !sfn.IsDir() {
			return ki.Continue
		}
		if sfn.DirRepo == nil {
			if !sfn.DetectVcsRepo(false) {
				return ki.Continue
			}
		}
		repo := sfn.DirRepo
		fmt.Printf("Updating %v repository: %s from: %s\n", repo.Vcs(), sfn.MyRelPath(), repo.Remote())
		err := repo.Update()
		if err != nil {
			fmt.Printf("error: %v\n", err)
		}
		return ki.Break
	})
}
