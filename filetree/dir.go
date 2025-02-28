// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// dirFlags are flags on directories: Open, SortBy, etc.
// These flags are stored in the DirFlagMap for persistence.
// This map is saved to a file, so these flags must be stored
// as bit flags instead of a struct to ensure efficient serialization.
type dirFlags int64 //enums:bitflag -trim-prefix dir

const (
	// dirIsOpen means directory is open -- else closed
	dirIsOpen dirFlags = iota

	// dirSortByName means sort the directory entries by name.
	// this overrides SortByModTime default on Tree if set.
	dirSortByName

	// dirSortByModTime means sort the directory entries by modification time.
	dirSortByModTime
)

// DirFlagMap is a map for encoding open directories and sorting preferences.
// The strings are typically relative paths. Map access is protected by Mutex.
type DirFlagMap struct {

	// map of paths and associated flags
	Map map[string]dirFlags

	// mutex for accessing map
	sync.Mutex
}

// init initializes the map, and sets the Mutex lock; must unlock manually.
func (dm *DirFlagMap) init() {
	dm.Lock()
	if dm.Map == nil {
		dm.Map = make(map[string]dirFlags)
	}
}

// isOpen returns true if path has isOpen bit flag set
func (dm *DirFlagMap) isOpen(path string) bool {
	dm.init()
	defer dm.Unlock()
	if df, ok := dm.Map[path]; ok {
		return df.HasFlag(dirIsOpen)
	}
	return false
}

// SetOpenState sets the given directory's open flag
func (dm *DirFlagMap) setOpen(path string, open bool) {
	dm.init()
	defer dm.Unlock()
	df := dm.Map[path]
	df.SetFlag(open, dirIsOpen)
	dm.Map[path] = df
}

// sortByName returns true if path is sorted by name (default if not in map)
func (dm *DirFlagMap) sortByName(path string) bool {
	dm.init()
	defer dm.Unlock()
	if df, ok := dm.Map[path]; ok {
		return df.HasFlag(dirSortByName)
	}
	return true
}

// sortByModTime returns true if path is sorted by mod time
func (dm *DirFlagMap) sortByModTime(path string) bool {
	dm.init()
	defer dm.Unlock()
	if df, ok := dm.Map[path]; ok {
		return df.HasFlag(dirSortByModTime)
	}
	return false
}

// setSortBy sets the given directory's sort by option
func (dm *DirFlagMap) setSortBy(path string, modTime bool) {
	dm.init()
	defer dm.Unlock()
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

// openPaths returns a list of open paths
func (dm *DirFlagMap) openPaths(root string) []string {
	dm.init()
	defer dm.Unlock()

	paths := make([]string, 0, len(dm.Map))
	for fn, df := range dm.Map {
		if !df.HasFlag(dirIsOpen) {
			continue
		}
		fpath := filepath.Join(root, fn)
		_, err := os.Stat(fpath)
		if err != nil {
			delete(dm.Map, fn)
			continue
		}
		rootClosed := false
		par := fn
		for {
			par, _ = filepath.Split(par)
			par = strings.TrimSuffix(par, "/")
			if par == "" || par == "." {
				break
			}
			if pdf, ook := dm.Map[par]; ook {
				if !pdf.HasFlag(dirIsOpen) {
					rootClosed = true
					break
				}
			}
		}
		if rootClosed {
			continue
		}
		paths = append(paths, fpath)
	}
	slices.Sort(paths)
	return paths
}
