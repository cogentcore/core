// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dirsort

import (
	"path/filepath"
	"slices"
	"sync"
	"time"

	"cogentcore.org/core/base/sharedfile"
)

// DataDir is the directory of the file of the sort orders that are [Set],
// which is shared by all apps: core sets it to the Cogent Core data
// directory. If it is empty, the sort orders are only kept in memory.
var DataDir string

// FileName is the name of the file of the sort orders in [DataDir].
const FileName = "dir-sort.json"

// MaxDirs is the maximum number of directories whose sort orders are
// saved: those set least recently are dropped beyond it.
var MaxDirs = 5000

// entry is a sort order of a directory in the file, with when it was set.
type entry struct {
	Sort

	// Time is when the sort order was set, in Unix seconds.
	Time int64
}

// store is the sort orders of the directories, in the file shared by all
// apps, which is read again when another process changes it.
type store struct {
	mu sync.Mutex

	// file is the file of the sort orders by directory.
	file sharedfile.File[map[string]entry]
}

// theStore is the store of the sort orders.
var theStore store

// path returns the path of the file, or "" if there is no [DataDir].
func (st *store) path() string {
	if DataDir == "" {
		return ""
	}
	return filepath.Join(DataDir, FileName)
}

// reload reads the file again if it changed since it was read (as another
// process saved it), or if [DataDir] changed, under the mutex. A file that
// is not valid is replaced when a sort order is set.
func (st *store) reload() {
	st.file.Path = st.path()
	st.file.Reload()
}

// get returns the sort order of the given (clean) directory.
func (st *store) get(dir string) (Sort, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.reload()
	e, ok := st.file.Data[dir]
	return e.Sort, ok
}

// set sets the sort order of the given (clean) directory, and saves the
// file, with it locked, after reading the changes of other processes.
func (st *store) set(dir string, s Sort) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.file.Path = st.path()
	return st.file.Update(func(dirs *map[string]entry) {
		if *dirs == nil {
			*dirs = map[string]entry{}
		}
		(*dirs)[dir] = entry{Sort: s, Time: time.Now().Unix()}
		prune(*dirs)
	})
}

// prune drops the sort orders set least recently beyond [MaxDirs].
func prune(dirs map[string]entry) {
	if len(dirs) <= MaxDirs {
		return
	}
	type dirTime struct {
		dir  string
		time int64
	}
	dts := make([]dirTime, 0, len(dirs))
	for d, e := range dirs {
		dts = append(dts, dirTime{d, e.Time})
	}
	slices.SortFunc(dts, func(a, b dirTime) int { return int(b.time - a.time) })
	for _, dt := range dts[MaxDirs:] {
		delete(dirs, dt.dir)
	}
}
