// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/Masterminds/vcs"
	"github.com/goki/gi/gi"
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

	// AddFile adds a path to a file to the cached list of files in repo - purely to limit disk access
	AddFile(path string)

	// Get is used to perform an initial clone/checkout of a repository.
	InRepo(filename string) bool

	// Add adds the file to the repo
	Add(filename string) error

	// Move moves updates the repo with the rename
	Move(oldpath, newpath string) error

	// Remove removes the file from the repo
	Remove(filename string) error

	// RemoveKeepLocal removes the file from the repo but keeps the file itself
	RemoveKeepLocal(filename string) error
}

func NewRepo(remote, local string) (Repo, error) {
	repo, err := vcs.NewRepo(remote, local)
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
	return nil, err
}

type GitRepo struct {
	vcs.Repo
	Files []string
}

// AddFile adds a path to a file to the cached list of files in repo - purely to limit disk access
func (gr GitRepo) AddFile(path string) {
	gi.StringsAppendIfUnique(&gr.Files, path, 500)
}

// IsTracked returns true if the file is being tracked in the specified repository
func (gr GitRepo) InRepo(filename string) bool {
	for _, f := range gr.Files {
		if f == filename {
			return true
		}
	}
	return false
}

// Add adds the file to the repo
func (gr GitRepo) Add(filename string) error {
	oscmd := exec.Command("git", "add", filename)
	stdoutStderr, err := oscmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s\n", err)
		return err
	}
	fmt.Printf("%s\n", stdoutStderr)
	return nil
}

// Move moves updates the repo with the rename
func (gr GitRepo) Move(oldpath, newpath string) error {
	oscmd := exec.Command("git", "mv", oldpath, newpath)
	stdoutStderr, err := oscmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s\n", err)
		return err
	}
	fmt.Printf("%s\n", stdoutStderr)
	return nil
}

// Remove removes the file from the repo
func (gr GitRepo) Remove(filename string) error {
	oscmd := exec.Command("git", "rm", filename)
	stdoutStderr, err := oscmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s\n", err)
		return err
	}
	fmt.Printf("%s\n", stdoutStderr)
	return nil
}

// Remove removes the file from the repo
func (gr GitRepo) RemoveKeepLocal(filename string) error {
	oscmd := exec.Command("git", "rm", "--cached", filename)
	stdoutStderr, err := oscmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s\n", err)
		return err
	}
	fmt.Printf("%s\n", stdoutStderr)
	return nil
}
