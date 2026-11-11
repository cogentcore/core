// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dirsort

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"
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

// store is the sort orders of the directories, as last read from the file
// (or set), and the state of the file when it was read, to read it again
// when another process changes it.
type store struct {
	mu sync.Mutex

	// dirs are the sort orders by directory.
	dirs map[string]entry

	// file is the path of the file that was read, which changes with
	// [DataDir] (in tests).
	file string

	// modTime and size are those of the file when it was read.
	modTime time.Time
	size    int64
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
// process saved it), under the mutex.
func (st *store) reload() {
	fn := st.path()
	if fn == "" {
		if st.dirs == nil {
			st.dirs = map[string]entry{}
		}
		return
	}
	info, err := os.Stat(fn)
	if err != nil {
		if st.dirs == nil || st.file != fn {
			st.dirs, st.file = map[string]entry{}, fn
		}
		return
	}
	if st.dirs != nil && st.file == fn && info.ModTime().Equal(st.modTime) && info.Size() == st.size {
		return
	}
	dirs := map[string]entry{}
	if b, err := os.ReadFile(fn); err == nil {
		// a file that is not valid is replaced when one is set
		json.Unmarshal(b, &dirs)
	}
	st.dirs, st.file = dirs, fn
	st.modTime, st.size = info.ModTime(), info.Size()
}

// get returns the sort order of the given (clean) directory.
func (st *store) get(dir string) (Sort, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.reload()
	e, ok := st.dirs[dir]
	return e.Sort, ok
}

// set sets the sort order of the given (clean) directory, and saves the
// file, with it locked, after reading the changes of other processes.
func (st *store) set(dir string, s Sort) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	fn := st.path()
	if fn == "" {
		st.reload()
		st.dirs[dir] = entry{Sort: s, Time: time.Now().Unix()}
		return nil
	}
	unlock, err := lockFile(fn + ".lck")
	if err != nil {
		return err
	}
	defer unlock()
	st.reload()
	st.dirs[dir] = entry{Sort: s, Time: time.Now().Unix()}
	st.prune()
	b, err := json.MarshalIndent(st.dirs, "", "\t")
	if err != nil {
		return err
	}
	// written whole, then renamed, so that other processes never read
	// a partial file
	tmp := fn + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, fn); err != nil {
		return err
	}
	if info, err := os.Stat(fn); err == nil {
		st.modTime, st.size = info.ModTime(), info.Size()
	}
	return nil
}

// prune drops the sort orders set least recently beyond [MaxDirs].
func (st *store) prune() {
	if len(st.dirs) <= MaxDirs {
		return
	}
	type dirTime struct {
		dir  string
		time int64
	}
	dts := make([]dirTime, 0, len(st.dirs))
	for d, e := range st.dirs {
		dts = append(dts, dirTime{d, e.Time})
	}
	slices.SortFunc(dts, func(a, b dirTime) int { return int(b.time - a.time) })
	for _, dt := range dts[MaxDirs:] {
		delete(st.dirs, dt.dir)
	}
}

// lockStale is how long a lock file can be held before it is taken as
// left by a process that ended while holding it.
const lockStale = 2 * time.Second

// lockFile locks the given lock file, by creating it (which only one
// process can do), waiting for another process to remove it, or taking it
// over when it is stale. It returns the function to unlock it.
func lockFile(fn string) (unlock func(), err error) {
	for range 100 {
		f, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			f.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
			f.Close()
			return func() { os.Remove(fn) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		if info, err := os.Stat(fn); err == nil && time.Since(info.ModTime()) > lockStale {
			os.Remove(fn)
			continue
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil, errors.New("dirsort: could not lock " + fn)
}
