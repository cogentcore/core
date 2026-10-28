// Copyright (c) 2018, The Cogent Core Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vcs provides a version control system interface for git and
// Subversion repositories, with file status information and operations,
// using the git and svn commands.
package vcs

//go:generate core generate

import (
	"errors"
	"path/filepath"

	"cogentcore.org/core/base/fsx"
)

type Types int32 //enums:enum -accept-lower

const (
	NoVCS Types = iota
	Git
	Svn
	Bzr
	Hg
)

// Repo is a version control repository, with support for
// file status information and operations.
type Repo interface {
	// LocalPath returns the path of the local repository (working copy).
	LocalPath() string

	// Remote returns the location of the remote repository (a URL),
	// or "" if there is none.
	Remote() string

	// Update updates the local repository from the remote (pull).
	Update() error

	// UpdateVersion checks out the given version (revision).
	UpdateVersion(version string) error

	// Current returns the current branch, or version if not on a branch.
	Current() (string, error)

	// Branches returns the list of branches.
	Branches() ([]string, error)

	// Type returns the type of repo we are using
	Type() Types

	// Files returns a map of the current files and their status,
	// using a cached version of the file list if available.
	// nil will be returned immediately if no cache is available.
	// The given onUpdated function will be called from a separate
	// goroutine when the updated list of the files is available
	// (an update is always triggered even if the function is nil).
	Files(onUpdated func(f Files)) (Files, error)

	// StatusFast returns file status based on the cached file info,
	// which might be slightly stale. Much faster than Status.
	// Returns Stored if no files have been cached yet (shown as normal).
	StatusFast(fname string) FileStatus

	// Status returns status of given file -- returns Untracked and error
	// message on any error. FileStatus is a summary status category,
	// and string return value is more detailed status information formatted
	// according to standard conventions of given VCS.
	Status(fname string) (FileStatus, string)

	// Add adds the file to the repo
	Add(fname string) error

	// Move moves the file using VCS command to keep it updated
	Move(oldpath, newpath string) error

	// Delete removes the file from the repo and working copy.
	// Uses "force" option to ensure deletion.
	Delete(fname string) error

	// DeleteRemote removes the file from the repo but keeps the local file itself
	DeleteRemote(fname string) error

	// CommitFile commits a single file
	CommitFile(fname string, message string) error

	// RevertFile reverts a single file to the version that it was last in VCS,
	// losing any local changes (destructive!)
	RevertFile(fname string) error

	// FileContents returns the contents of given file, as a []byte array
	// at given revision specifier, or the current file in the working copy
	// if empty (use HEAD for the last commit).
	// -1, -2 etc also work as universal ways of specifying prior revisions.
	FileContents(fname string, rev string) ([]byte, error)

	// Log returns the log history of commits for given filename
	// (or all files if empty).  If since is non-empty, it should be
	// a date-like expression that the VCS will understand, such as
	// 1/1/2020, yesterday, last year, etc.  SVN only understands a
	// number as a maximum number of items to return.
	Log(fname string, since string) (Log, error)

	// CommitDesc returns the full textual description of the given commit,
	// if rev is empty, defaults to current HEAD, -1, -2 etc also work as universal
	// ways of specifying prior revisions.
	// Optionally includes diffs for the changes (otherwise just a list of files
	// with modification status).
	CommitDesc(rev string, diffs bool) ([]byte, error)

	// FilesChanged returns the list of files changed and their statuses,
	// between two revisions.
	// If revA is empty, defaults to current HEAD; revB defaults to HEAD-1.
	// -1, -2 etc also work as universal ways of specifying prior revisions.
	// Optionally includes diffs for the changes.
	FilesChanged(revA, revB string, diffs bool) ([]byte, error)

	// Blame returns an annotated report about the file, showing which revision last
	// modified each line.
	Blame(fname string) ([]byte, error)
}

// NewRepo returns a [Repo] for the repository at the given local path,
// of the type found by [DetectRepo]. The remote is the location (URL) of
// the remote repository, or for git, the name of a remote (origin if empty).
func NewRepo(remote, local string) (Repo, error) {
	switch DetectRepo(local) {
	case Git:
		return NewGitRepo(remote, local)
	case Svn:
		return NewSvnRepo(remote, local)
	}
	return nil, errors.New("no supported version control repository at: " + local)
}

// DetectRepo attempts to detect the presence of a repository at the given
// directory path -- returns type of repository if found, else NoVCS.
// Very quickly just looks for signature file name:
// .git for git
// .svn for svn -- but note that this will find any subdir in svn rep.o
func DetectRepo(path string) Types {
	if fsx.HasFile(path, ".git") {
		return Git
	}
	if fsx.HasFile(path, ".svn") {
		return Svn
	}
	// todo: rest later..
	return NoVCS
}

// DetectRepoRoot returns the root directory and type of the repository
// containing the given directory path: the first one with a repository
// found by [DetectRepo], starting from the path itself and going up.
// It returns "", NoVCS if there is none.
func DetectRepoRoot(path string) (string, Types) {
	dir, err := filepath.Abs(path)
	if err != nil {
		return "", NoVCS
	}
	for {
		if typ := DetectRepo(dir); typ != NoVCS {
			return dir, typ
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", NoVCS
		}
		dir = parent
	}
}

// relPath return the path relative to the repository LocalPath()
func relPath(repo Repo, path string) string {
	relpath, _ := filepath.Rel(repo.LocalPath(), path)
	return relpath
}
