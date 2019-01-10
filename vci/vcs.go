// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

import (
	"errors"
	"fmt"

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
			r := &SvnRepo{}
			r.Repo = repo
			return r, err
		case vcs.Hg:
			err = fmt.Errorf("Hg version control not yet supported")
		case vcs.Bzr:
			err = fmt.Errorf("Bzr version control not yet supported")
		}
	}
	return nil, err
}
