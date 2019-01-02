// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Masterminds/vcs"
)

var (
	// ErrUnknownVCS is returned when VCS cannot be determined from the vcs Repo
	ErrUnknownVCS = errors.New("Unknown VCS")
)

// Repo provides an interface that parallels vcs.Repo (https://github.com/Masterminds/vcs)
// with some additional functions
type Repo interface {
	// vcs.Repo includes those interface functions
	vcs.Repo

	// CacheFileNames reads all of the file names stored in the repository
	// into a local cache to speed checking whether a file is in the repository or not.
	CacheFileNames()

	// CacheFilesModified gets a list of files in repository changed since last commit
	CacheFilesModified()

	// CacheFilesAdded gets a list of files added to repository but not yet committed
	CacheFilesAdded()

	// CacheRefresh calls all of the Cache functions
	CacheRefresh()

	// InRepo returns true if filename is in the repository -- uses CacheFileNames --
	// will do that automatically but if cache might be stale, call it to refresh
	InRepo(filename string) bool

	// IsModified checks against the cached list to see if the file is modified since the last commit
	// IsDirty() will check with the repo rather than the cached list
	IsModified(filename string) bool

	// IsAdded checks for the file in the cached FilesAdded list
	IsAdded(filename string) bool

	// Add adds the file to the repo
	Add(filename string) error

	// Move moves updates the repo with the rename
	Move(oldpath, newpath string) error

	// Remove removes the file from the repo
	Remove(filename string) error

	// RemoveKeepLocal removes the file from the repo but keeps the file itself
	RemoveKeepLocal(filename string) error

	// CommitFile commits a single file
	CommitFile(filename string, message string) error

	// RevertFile reverts a single file
	RevertFile(filename string) error
}

func NewRepo(remote, local string) (Repo, error) {
	repo, err := vcs.NewRepo(remote, local)
	if err == nil {
		switch repo.Vcs() {
		case vcs.Git:
			r := &GitRepo{}
			r.Repo = repo
			return r, err
		case vcs.Svn:
			fmt.Println("Svn version control not yet supported ")
		case vcs.Hg:
			fmt.Println("Hg version control not yet supported ")
		case vcs.Bzr:
			fmt.Println("Bzr version control not yet supported ")
		}
	}
	return nil, err
}

type GitRepo struct {
	vcs.Repo
	FilesAll      map[string]struct{}
	FilesModified map[string]struct{}
	FilesAdded    map[string]struct{}
}

func (gr *GitRepo) CacheFileNames() {
	gr.FilesAll = make(map[string]struct{}, 1000)
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = gr.LocalPath()
	bytes, _ := cmd.Output()
	sep := byte(10) // Linefeed is the separator - will this work cross platform?
	names := strings.Split(string(bytes), string(sep))
	for _, n := range names {
		gr.FilesAll[n] = struct{}{}
	}
}

func (gr *GitRepo) CacheFilesModified() {
	gr.FilesModified = make(map[string]struct{}, 100)
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=M", "HEAD")
	cmd.Dir = gr.LocalPath()
	bytes, _ := cmd.Output()
	sep := byte(10) // Linefeed is the separator - will this work cross platform?
	names := strings.Split(string(bytes), string(sep))
	for _, n := range names {
		gr.FilesModified[n] = struct{}{}
	}
}

func (gr *GitRepo) CacheFilesAdded() {
	gr.FilesAdded = make(map[string]struct{}, 100)
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=A", "HEAD")
	cmd.Dir = gr.LocalPath()
	bytes, _ := cmd.Output()
	sep := byte(10) // Linefeed is the separator - will this work cross platform?
	names := strings.Split(string(bytes), string(sep))
	for _, n := range names {
		gr.FilesAdded[n] = struct{}{}
	}
}

func (gr *GitRepo) CacheRefresh() {
	gr.CacheFileNames()
	gr.CacheFilesAdded()
	gr.CacheFilesModified()
}

func (gr *GitRepo) InRepo(filename string) bool {
	if len(gr.FilesAll) == 0 {
		gr.CacheFileNames()
	}
	_, has := gr.FilesAll[filename]
	return has
}

func (gr *GitRepo) IsModified(filename string) bool {
	if len(gr.FilesModified) == 0 {
		gr.CacheFilesModified()
	}
	_, has := gr.FilesModified[filename]
	return has
}

func (gr *GitRepo) IsAdded(filename string) bool {
	if len(gr.FilesAdded) == 0 {
		gr.CacheFilesAdded()
	}
	_, has := gr.FilesAdded[filename]
	return has
}

// Add adds the file to the repo
func (gr *GitRepo) Add(filename string) error {
	oscmd := exec.Command("git", "add", filename)
	stdoutStderr, err := oscmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s\n", stdoutStderr)
		return err
	}
	gr.CacheFilesAdded()
	return nil
}

// Move moves updates the repo with the rename
func (gr *GitRepo) Move(oldpath, newpath string) error {
	oscmd := exec.Command("git", "mv", oldpath, newpath)
	stdoutStderr, err := oscmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s\n", stdoutStderr)
		return err
	}
	return nil
}

// Remove removes the file from the repo
func (gr *GitRepo) Remove(filename string) error {
	oscmd := exec.Command("git", "rm", filename)
	stdoutStderr, err := oscmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s\n", stdoutStderr)
		return err
	}
	gr.CacheRefresh()
	return nil
}

// Remove removes the file from the repo
func (gr *GitRepo) RemoveKeepLocal(filename string) error {
	oscmd := exec.Command("git", "rm", "--cached", filename)
	stdoutStderr, err := oscmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s\n", stdoutStderr)
		return err
	}
	gr.CacheRefresh()
	return nil
}

// CommitFile commits single file to repo staging
func (gr *GitRepo) CommitFile(filename string, message string) error {
	oscmd := exec.Command("git", "commit", filename, "-m", message)
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", stdoutStderr)
		return err
	}
	gr.CacheRefresh()
	return nil
}

// RevertFile reverts a single file to last commit of master
func (gr *GitRepo) RevertFile(filename string) error {
	oscmd := exec.Command("git", "checkout", filename)
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", stdoutStderr)
		return err
	}
	gr.CacheFilesModified()
	return nil
}
