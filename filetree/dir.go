// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

// dirFlags are flags on directories: Open.
// These flags are stored in the DirFlagMap for persistence.
// This map is saved to a file, so these flags must be stored
// as bit flags instead of a struct to ensure efficient serialization.
type dirFlags int32 //enums:bitflag -trim-prefix dir

const (
	// dirIsOpen means directory is open -- else closed
	dirIsOpen dirFlags = iota

	// dirSortByName is no longer used: the sort order of a directory is
	// that of [dirsort.For]. It is kept for the flags that were saved.
	dirSortByName

	// dirSortByModTime is no longer used: the sort order of a directory is
	// that of [dirsort.For]. It is kept for the flags that were saved.
	dirSortByModTime
)

// DirFlagMap is a map for encoding open directories.
// The strings are typically relative paths. Map access is protected by Mutex.
type DirFlagMap struct {

	// map of paths and associated flags
	Map map[string]dirFlags

	// mutex for accessing map
	sync.Mutex
}

// Copy copies flag map from another map, safely avoiding copying the mutex.
func (dm *DirFlagMap) Copy(cp *DirFlagMap) {
	dm.init()
	defer dm.Unlock()
	if cp.Map == nil {
		return
	}
	maps.Copy(dm.Map, cp.Map)
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

// openPaths returns a list of open paths
func (dm *DirFlagMap) openPaths(root string) []string {
	dm.init()
	defer dm.Unlock()

	var paths []string
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
			par = filepath.Dir(par)
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
