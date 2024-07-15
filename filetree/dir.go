// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import "sync"

// dirFlags are flags on directories: Open, SortBy, etc.
// These flags are stored in the DirFlagMap for persistence.
// This map is saved to a file, so these flags must be stored
// as bit flags instead of a struct to ensure efficient serialization.
type dirFlags int64 //enums:bitflag -trim-prefix dir

const (
	// dirIsOpen means directory is open -- else closed
	dirIsOpen dirFlags = iota

	// dirSortByName means sort the directory entries by name.
	// this is mutex with other sorts -- keeping option open for non-binary sort choices.
	dirSortByName

	// dirSortByModTime means sort the directory entries by modification time
	dirSortByModTime
)

// DirFlagMap is a map for encoding directories that are open in the file
// tree.  The strings are typically relative paths.  The bool value is used to
// mark active paths and inactive (unmarked) ones can be removed.
// Map access is protected by Mutex.
type DirFlagMap struct {

	// map of paths and associated flags
	Map map[string]dirFlags

	// mutex for accessing map
	mu sync.Mutex
}

// init initializes the map, and sets the Mutex lock -- must unlock manually
func (dm *DirFlagMap) init() {
	dm.mu.Lock()
	if dm.Map == nil {
		dm.Map = make(map[string]dirFlags)
	}
}

// isOpen returns true if path has isOpen bit flag set
func (dm *DirFlagMap) isOpen(path string) bool {
	dm.init()
	defer dm.mu.Unlock()
	if df, ok := dm.Map[path]; ok {
		return df.HasFlag(dirIsOpen)
	}
	return false
}

// SetOpenState sets the given directory's open flag
func (dm *DirFlagMap) setOpen(path string, open bool) {
	dm.init()
	defer dm.mu.Unlock()
	df := dm.Map[path]
	df.SetFlag(open, dirIsOpen)
	dm.Map[path] = df
}

// sortByName returns true if path is sorted by name (default if not in map)
func (dm *DirFlagMap) sortByName(path string) bool {
	dm.init()
	defer dm.mu.Unlock()
	if df, ok := dm.Map[path]; ok {
		return df.HasFlag(dirSortByName)
	}
	return true
}

// sortByModTime returns true if path is sorted by mod time
func (dm *DirFlagMap) sortByModTime(path string) bool {
	dm.init()
	defer dm.mu.Unlock()
	if df, ok := dm.Map[path]; ok {
		return df.HasFlag(dirSortByModTime)
	}
	return false
}

// setSortBy sets the given directory's sort by option
func (dm *DirFlagMap) setSortBy(path string, modTime bool) {
	dm.init()
	defer dm.mu.Unlock()
	df := dm.Map[path]
	if modTime {
		df.SetFlag(true, dirSortByModTime)
		df.SetFlag(false, dirSortByName)
	} else {
		df.SetFlag(false, dirSortByModTime)
		df.SetFlag(true, dirSortByName)
	}
	dm.Map[path] = df
}
