// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/glop/dirs"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/textbuf"
	"cogentcore.org/core/vci"
	"github.com/Masterminds/vcs"
)

// FirstVCS returns the first VCS repository starting from this node and going down.
// also returns the node having that repository
func (fn *Node) FirstVCS() (vci.Repo, *Node) {
	var repo vci.Repo
	var rnode *Node
	fn.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		sfn := AsNode(wi)
		if sfn == nil {
			return ki.Continue
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

// DetectVCSRepo detects and configures DirRepo if this directory is root of
// a VCS repository.  if updateFiles is true, gets the files in the dir.
// returns true if a repository was newly found here.
func (fn *Node) DetectVCSRepo(updateFiles bool) bool {
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

// AddToVCSSel adds selected files to version control system
func (fn *Node) AddToVCSSel() { //gti:add
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.AddToVCS()
	}
}

// AddToVCS adds file to version control
func (fn *Node) AddToVCS() {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	// fmt.Printf("adding to vcs: %v\n", fn.FPath)
	err := repo.Add(string(fn.FPath))
	if err == nil {
		fn.Info.Vcs = vci.Added
		fn.SetNeedsRender(true)
		return
	}
	fmt.Println(err)
}

// DeleteFromVCSSel removes selected files from version control system
func (fn *Node) DeleteFromVCSSel() { //gti:add
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.DeleteFromVCS()
	}
}

// DeleteFromVCS removes file from version control
func (fn *Node) DeleteFromVCS() {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	// fmt.Printf("deleting remote from vcs: %v\n", fn.FPath)
	err := repo.DeleteRemote(string(fn.FPath))
	if fn != nil && err == nil {
		fn.Info.Vcs = vci.Deleted
		fn.SetNeedsRender(true)
		return
	}
	fmt.Println(err)
}

// CommitToVCSSel commits to version control system based on last selected file
func (fn *Node) CommitToVCSSel() { //gti:add
	sels := fn.SelectedViews()
	n := len(sels)
	if n == 0 { // shouldn't happen
		return
	}
	sn := AsNode(sels[n-1])
	giv.CallFunc(sn, fn.CommitToVCS)
}

// CommitToVCS commits file changes to version control system
func (fn *Node) CommitToVCS(message string) (err error) {
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
	fn.SetNeedsRender(true)
	return err
}

// RevertVCSSel removes selected files from version control system
func (fn *Node) RevertVCSSel() { //gti:add
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.RevertVCS()
	}
}

// RevertVCS reverts file changes since last commit
func (fn *Node) RevertVCS() (err error) {
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
	fn.SetNeedsRender(true)
	return err
}

// DiffVCSSel shows the diffs between two versions of selected files, given by the
// revision specifiers -- if empty, defaults to A = current HEAD, B = current WC file.
// -1, -2 etc also work as universal ways of specifying prior revisions.
// Diffs are shown in a DiffViewDialog.
func (fn *Node) DiffVCSSel(rev_a string, rev_b string) { //gti:add
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.DiffVCS(rev_a, rev_b)
	}
}

// DiffVCS shows the diffs between two versions of this file, given by the
// revision specifiers -- if empty, defaults to A = current HEAD, B = current WC file.
// -1, -2 etc also work as universal ways of specifying prior revisions.
// Diffs are shown in a DiffViewDialog.
func (fn *Node) DiffVCS(rev_a, rev_b string) error {
	repo, _ := fn.Repo()
	if repo == nil {
		return errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	if fn.Info.Vcs == vci.Untracked {
		return errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	_, err := texteditor.DiffViewDialogFromRevs(fn, repo, string(fn.FPath), fn.Buf, rev_a, rev_b)
	return err
}

// LogVCSSel shows the VCS log of commits for selected files, optionally with a
// since date qualifier: If since is non-empty, it should be
// a date-like expression that the VCS will understand, such as
// 1/1/2020, yesterday, last year, etc.  SVN only understands a
// number as a maximum number of items to return.
// If allFiles is true, then the log will show revisions for all files, not just
// this one.
// Returns the Log and also shows it in a VCSLogView which supports further actions.
func (fn *Node) LogVCSSel(allFiles bool, since string) { //gti:add
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.LogVCS(allFiles, since)
	}
}

// LogVCS shows the VCS log of commits for this file, optionally with a
// since date qualifier: If since is non-empty, it should be
// a date-like expression that the VCS will understand, such as
// 1/1/2020, yesterday, last year, etc.  SVN only understands a
// number as a maximum number of items to return.
// If allFiles is true, then the log will show revisions for all files, not just
// this one.
// Returns the Log and also shows it in a VCSLogView which supports further actions.
func (fn *Node) LogVCS(allFiles bool, since string) (vci.Log, error) {
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
	VCSLogViewDialog(nil, repo, lg, fnm, since)
	return lg, nil
}

// BlameVCSSel shows the VCS blame report for this file, reporting for each line
// the revision and author of the last change.
func (fn *Node) BlameVCSSel() { //gti:add
	sels := fn.SelectedViews()
	n := len(sels)
	for i := n - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.BlameVCS()
	}
}

// BlameDialog opens a dialog for displaying VCS blame data using textview.TwinViews.
// blame is the annotated blame code, while fbytes is the original file contents.
func BlameDialog(ctx gi.Widget, fname string, blame, fbytes []byte) *texteditor.TwinEditors {
	title := "VCS Blame: " + dirs.DirAndFile(fname)

	d := gi.NewBody().AddTitle(title)
	tv := texteditor.NewTwinEditors(d, "twin-view")
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
	tv.SetSplits(.3, .7)

	tva, tvb := tv.Editors()
	tva.Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePre
		s.Min.X.Ch(30)
		s.Min.Y.Em(40)
	})
	tvb.Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePre
		s.Min.X.Ch(80)
		s.Min.Y.Em(40)
	})
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddOk(pw)
	})
	d.NewFullDialog(ctx).SetNewWindow(true).Run()
	return tv
}

// BlameVCS shows the VCS blame report for this file, reporting for each line
// the revision and author of the last change.
func (fn *Node) BlameVCS() ([]byte, error) {
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

// UpdateAllVCS does an update on any repositories below this one in file tree
func (fn *Node) UpdateAllVCS() {
	fn.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		sfn := AsNode(wi)
		if sfn == nil {
			return ki.Continue
		}
		if !sfn.IsDir() {
			return ki.Continue
		}
		if sfn.DirRepo == nil {
			if !sfn.DetectVCSRepo(false) {
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

////////////////////////////////////////////////////////////////////////////////////////
//  VersCtrlValue

// VersCtrlSystems is a list of supported Version Control Systems.
// These must match the VCS Types from goki/pi/vci which in turn
// is based on masterminds/vcs
var VersCtrlSystems = []string{"git", "svn", "bzr", "hg"}

// IsVersCtrlSystem returns true if the given string matches one of the
// standard VersCtrlSystems -- uses lowercase version of str.
func IsVersCtrlSystem(str string) bool {
	stl := strings.ToLower(str)
	for _, vcn := range VersCtrlSystems {
		if stl == vcn {
			return true
		}
	}
	return false
}

// VersCtrlName is the name of a version control system
type VersCtrlName string

func VersCtrlNameProper(vc string) VersCtrlName {
	vcl := strings.ToLower(vc)
	for _, vcnp := range VersCtrlSystems {
		vcnpl := strings.ToLower(vcnp)
		if strings.Compare(vcl, vcnpl) == 0 {
			return VersCtrlName(vcnp)
		}
	}
	return ""
}

// Value registers VersCtrlValue as the viewer of VersCtrlName
func (kn VersCtrlName) Value() giv.Value {
	return &VersCtrlValue{}
}

// VersCtrlValue presents an action for displaying an VersCtrlName and selecting
// from StringPopup
type VersCtrlValue struct {
	giv.ValueBase
}

func (vv *VersCtrlValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *VersCtrlValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	bt.SetTextUpdate(txt)
}

func (vv *VersCtrlValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config()
	bt.OnClick(func(e events.Event) {
		if !vv.IsReadOnly() {
			vv.OpenDialog(bt, nil)
		}
	})
	vv.UpdateWidget()
}

func (vv *VersCtrlValue) HasDialog() bool { return true }

func (vv *VersCtrlValue) OpenDialog(ctx gi.Widget, fun func()) {
	cur := laser.ToString(vv.Value.Interface())
	m := gi.NewMenuFromStrings(VersCtrlSystems, cur, func(index int) {
		vv.SetValue(VersCtrlSystems[index])
		vv.UpdateWidget()
	})
	gi.NewMenuStage(m, ctx, ctx.ContextMenuPos(nil)).Run()
}
