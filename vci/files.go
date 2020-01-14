// Copyright (c) 2020, The GoGi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

import "github.com/goki/ki/kit"

// Files is a map used for storing files in a repository along with their status
type Files map[string]FileStatus

// Status returns the VCS file status associated with given filename,
// returning Untracked if not found and safe to empty map.
func (fl *Files) Status(fname string) FileStatus {
	if *fl == nil || len(*fl) == 0 {
		return Untracked
	}
	st, ok := (*fl)[fname]
	if !ok {
		return Untracked
	}
	return st
}

// FileStatus indicates the status of files in the repository
type FileStatus int32

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

	FileStatusN
)

//go:generate stringer -type=FileStatus

var KiT_FileStatus = kit.Enums.AddEnum(FileStatusN, kit.NotBitFlag, nil)

func (ev FileStatus) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *FileStatus) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }
