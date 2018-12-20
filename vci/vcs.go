// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

import (
	"errors"
	"fmt"
	"log"
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

	// InRepo returns true if filename is in the repository -- must have called CacheFileNames
	// first
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
	Files map[string]struct{}
}

func (gr *GitRepo) CacheFileNames() {
	gr.Files = make(map[string]struct{}, 1000)
	bytes, _ := exec.Command("git", "ls-files", gr.LocalPath()).Output()
	sep := byte(10) // ??
	names := strings.Split(string(bytes), string(sep))
	for _, n := range names {
		gr.Files[n] = struct{}{}
	}
}

func (gr *GitRepo) InRepo(filename string) bool {
	if len(gr.Files) == 0 {
		log.Println("GitRepo: must call CacheFileNames before using InRepo check")
		return false
	}
	_, has := gr.Files[filename]
	return has
}

// Add adds the file to the repo
func (gr *GitRepo) Add(filename string) error {
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
func (gr *GitRepo) Move(oldpath, newpath string) error {
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
func (gr *GitRepo) Remove(filename string) error {
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
func (gr *GitRepo) RemoveKeepLocal(filename string) error {
	oscmd := exec.Command("git", "rm", "--cached", filename)
	stdoutStderr, err := oscmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s\n", err)
		return err
	}
	fmt.Printf("%s\n", stdoutStderr)
	return nil
}
