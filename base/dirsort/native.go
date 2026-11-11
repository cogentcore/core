// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dirsort

import (
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Native returns the sort order that the file manager of the operating
// system uses for the given directory, and whether there is one: that of
// the directory, or else the default of the file manager.
//
//   - macOS: the Finder, from the view settings in the .DS_Store file of the
//     directory, or else its standard view settings (see [finderSort]).
//   - Linux: Dolphin (KDE), from the .directory file of the directory, and
//     Nautilus (GNOME), from the metadata of the directory, or else their
//     defaults (see [linuxSort]).
//   - Windows: none, as Explorer keeps the views of folders in the
//     registry, in binary records ("shell bags") that are not documented.
var Native = func(dir string) (Sort, bool) {
	dir = cleanDir(dir)
	switch runtime.GOOS {
	case "darwin":
		return finderSort(dir)
	case "linux", "freebsd", "openbsd", "netbsd":
		return linuxSort(dir)
	}
	return Sort{}, false
}

// cache caches values for a short time, as the native sort orders, which
// can take running a program to get, of the directories that are read
// again as they change.
type cache[V any] struct {
	mu      sync.Mutex
	entries map[string]cacheEntry[V]
}

// cacheEntry is a value in a [cache], with when it was got.
type cacheEntry[V any] struct {
	v    V
	time time.Time
}

// cacheTime is how long values are cached.
const cacheTime = 5 * time.Second

// get returns the cached value of the given key, or else the value of the
// given function for it, which is cached.
func (c *cache[V]) get(key string, f func() V) V {
	c.mu.Lock()
	e, ok := c.entries[key]
	c.mu.Unlock()
	if ok && time.Since(e.time) < cacheTime {
		return e.v
	}
	v := f()
	c.mu.Lock()
	if c.entries == nil || len(c.entries) > 1000 {
		c.entries = map[string]cacheEntry[V]{}
	}
	c.entries[key] = cacheEntry[V]{v, time.Now()}
	c.mu.Unlock()
	return v
}

// nativeSort is a sort order of a native file manager, and whether it has
// one, as cached.
type nativeSort struct {
	s  Sort
	ok bool
}

// homeDir returns the home directory of the user, or "".
func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

// byField returns the field of the given name of a column of a file
// manager (as dateModified for the Finder, or date_modified for Nautilus),
// ignoring case and _, and whether it is one: the times of creation,
// addition and opening are sorted by that of modification.
func byField(name string) (Fields, bool) {
	switch strings.ToLower(strings.ReplaceAll(name, "_", "")) {
	case "name", "text", "filename":
		return Name, true
	case "size", "physicalsize", "logicalsize":
		return Size, true
	case "kind", "type":
		return Kind, true
	case "datemodified", "modificationdate", "modificationtime", "mtime", "modified",
		"datecreated", "creationtime", "btime", "dateadded",
		"datelastopened", "dateaccessed", "accesstime", "atime", "recency":
		return ModTime, true
	}
	return 0, false
}

// downByDefault returns whether the given field is sorted down by default,
// as file managers do when they arrange by it: newest and largest first.
func downByDefault(by Fields) bool {
	return by == ModTime || by == Size
}
