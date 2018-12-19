// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vcsgi

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/goki/gi/gi"

	"github.com/Masterminds/vcs"
)

// A list of files that are tracked by version control - used temporarily as each directory is traversed
// VCS add/delete should set fn.InVc to keep the FileNode up to date
var VcsFiles []string

var (
	// ErrUnknownVCS is returned when VCS cannot be determined from the vcs Repo
	ErrUnknownVCS = errors.New("Unknown VCS")
)

func AppendToVcsFiles(str string) {
	gi.StringsAppendIfUnique(&VcsFiles, str, 500)
}

func InRepo(filename string) bool {
	for _, f := range VcsFiles {
		if f == filename {
			return true
		}
	}
	return false
}

// Repo provides an interface that parallels vcs.Repo (https://github.com/Masterminds/vcs)
// with some additional functions
type GiRepo interface {
	vcs.Repo

	// Get is used to perform an initial clone/checkout of a repository.
	IsTracked(filename string) bool

	// Add adds the file to the repo
	Add(filename string) error

	// Move moves updates the repo with the rename
	Move(oldpath, newpath string) error

	// Remove removes the file from the repo
	Remove(filename string) error

	// RemoveKeepLocal removes the file from the repo but keeps the file itself
	RemoveKeepLocal(filename string) error
}

type GitGiRepo struct {
	vcs.Repo
}

// IsTracked returns true if the file is being tracked in the specified repository
func (gr GitGiRepo) IsTracked(filename string) bool {
	for _, f := range VcsFiles {
		if f == filename {
			return true
		}
	}
	return false
}

// Add adds the file to the repo
func (gr GitGiRepo) Add(filename string) error {
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
func (gr GitGiRepo) Move(oldpath, newpath string) error {
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
func (gr GitGiRepo) Remove(filename string) error {
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
func (gr GitGiRepo) RemoveKeepLocal(filename string) error {
	oscmd := exec.Command("git", "rm", "--cached", filename)
	stdoutStderr, err := oscmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s\n", err)
		return err
	}
	fmt.Printf("%s\n", stdoutStderr)
	return nil
}
