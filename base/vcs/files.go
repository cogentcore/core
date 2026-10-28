// Copyright (c) 2020, The Cogent Core Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vcs

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
)

// Files is a map used for storing files in a repository along with their status
type Files map[string]FileStatus

// Status returns the VCS file status associated with given filename,
// returning Untracked if not found and safe to empty map.
func (fl *Files) Status(repo Repo, fname string) FileStatus {
	if *fl == nil || len(*fl) == 0 {
		return Untracked
	}
	st, ok := (*fl)[relPath(repo, fname)]
	if !ok {
		return Untracked
	}
	return st
}

// moved updates the files for the file or directory at the relative path
// oldrel being moved (renamed) to newrel in the repository: their status is
// Added, as the rename is staged, unless they are untracked.
// It does nothing for a nil map.
func (fl Files) moved(oldrel, newrel string) {
	if fl == nil {
		return
	}
	prefix := oldrel + string(filepath.Separator)
	moved := Files{}
	for f, st := range fl {
		if f != oldrel && !strings.HasPrefix(f, prefix) {
			continue
		}
		delete(fl, f)
		if st != Untracked {
			st = Added
		}
		moved[newrel+f[len(oldrel):]] = st
	}
	maps.Copy(fl, moved)
}

// allFiles returns a slice of all the files, recursively, within a given directory
func allFiles(path string) ([]string, error) {
	var fnms []string
	er := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fnms = append(fnms, path)
		return nil
	})
	return fnms, er
}

// FileStatus indicates the status of files in the repository
type FileStatus int32 //enums:enum

const (
	// Untracked means file is not under VCS control
	Untracked FileStatus = iota

	// Stored means file is stored under VCS control, and has not been modified in working copy
	Stored

	// Modified means file is under VCS control, and has been modified in working copy
	Modified

	// Added means file has just been added to VCS but is not yet committed
	Added

	// Deleted means file has been deleted from VCS
	Deleted

	// Conflicted means file is in conflict -- has not been merged
	Conflicted

	// Updated means file has been updated in the remote but not locally
	Updated
)
