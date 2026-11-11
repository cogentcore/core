// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dirsort provides the sort order of the files in each directory,
// shared by all of the apps (and all of their processes) that show
// directories: [For] returns the sort order of a directory, which is the
// one last [Set] for it in any app, or else the one that the file manager
// of the operating system uses for it (the Finder on macOS, from its
// .DS_Store files, and Dolphin or Nautilus on Linux; see [Native]), or
// else [Default]. The sort orders that are set are saved in a file in
// [DataDir], which is locked for the changes of each process, and read
// again when another process has changed it.
package dirsort

//go:generate core generate

import (
	"cmp"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/fileinfo"
)

// Fields are the fields of the files in a directory that they can be
// sorted by, which are named as the fields of [fileinfo.FileInfo].
type Fields int32 //enums:enum

const (
	// Name sorts the files by their names.
	Name Fields = iota

	// Size sorts the files by their sizes.
	Size

	// Kind sorts the files by their kinds (as Folder, or Code: Go).
	Kind

	// ModTime sorts the files by the time that they were last modified.
	ModTime
)

// Sort is a sort order of the files in a directory: by the given field, up
// (ascending: A to Z, smallest or oldest first) or down (descending).
type Sort struct { //types:add

	// By is the field that the files are sorted by.
	By Fields

	// Down is whether the files are sorted down (descending: Z to A,
	// largest or newest first), instead of up (ascending).
	Down bool
}

// Default is the sort order of a directory that has none (see [For]): by
// name, up.
var Default = Sort{}

// String returns the sort order as the name of its field and :up or :down,
// as [core.Table.SortFieldName] does.
func (s Sort) String() string {
	if s.Down {
		return s.By.String() + ":down"
	}
	return s.By.String() + ":up"
}

// Parse returns the sort order of the given string, as returned by
// [Sort.String] (where :up is the default), and whether it is one.
func Parse(str string) (Sort, bool) {
	name, dir, _ := strings.Cut(str, ":")
	var s Sort
	if err := s.By.SetString(name); err != nil {
		return Sort{}, false
	}
	switch dir {
	case "", "up":
	case "down":
		s.Down = true
	default:
		return Sort{}, false
	}
	return s, true
}

// For returns the sort order of the given directory: the one last [Set]
// for it, by any app, or else the one of the file manager of the operating
// system (see [Native]), or else [Default].
func For(dir string) Sort {
	if s, ok := Saved(dir); ok {
		return s
	}
	if Native != nil {
		if s, ok := Native(dir); ok {
			return s
		}
	}
	return Default
}

// Saved returns the sort order last [Set] for the given directory, by any
// app, and whether there is one.
func Saved(dir string) (Sort, bool) {
	return theStore.get(cleanDir(dir))
}

// Set sets the sort order of the given directory, for all apps, saving it
// in the file shared by them (see [DataDir]).
func Set(dir string, s Sort) error {
	return theStore.set(cleanDir(dir), s)
}

// cleanDir returns the absolute, clean path of the given directory, with
// its symbolic links evaluated (so that all of the paths of a directory
// have its sort order), by which its sort order is saved.
func cleanDir(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	return filepath.Clean(dir)
}

// Compare compares the given files in the sort order, returning -1 if a
// comes before b, and 1 if after, for [slices.SortFunc]. Files that are
// equal in the field of the sort order are in the order of their names
// (in the direction of the sort order).
func (s Sort) Compare(a, b *fileinfo.FileInfo) int {
	return s.compare(a.Name, b.Name, func() int {
		switch s.By {
		case Size:
			return cmp.Compare(a.Size, b.Size)
		case Kind:
			return compareNames(a.Kind, b.Kind)
		case ModTime:
			return a.ModTime.Compare(b.ModTime)
		}
		return 0
	})
}

// compare returns the comparison of files of the given names, in the sort
// order, given the comparison of their fields (by the sort order).
func (s Sort) compare(aName, bName string, field func() int) int {
	c := 0
	if s.By != Name {
		c = field()
	}
	if c == 0 {
		c = compareNames(aName, bName)
	}
	if s.Down {
		return -c
	}
	return c
}

// compareNames compares the given names as file managers do: ignoring case,
// and then by case.
func compareNames(a, b string) int {
	if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

// SortFiles sorts the given files in the sort order.
func (s Sort) SortFiles(files []*fileinfo.FileInfo) {
	slices.SortStableFunc(files, s.Compare)
}

// SortInfos sorts the given standard file infos in the sort order, as
// those of a directory listing. Their kinds are those of their names (see
// [fileinfo.FileInfo.MimeFromFilename]), without reading the files.
func (s Sort) SortInfos(infos []fs.FileInfo) {
	var kinds map[fs.FileInfo]string
	if s.By == Kind {
		kinds = make(map[fs.FileInfo]string, len(infos))
		for _, info := range infos {
			kinds[info] = kindOf(info)
		}
	}
	slices.SortStableFunc(infos, func(a, b fs.FileInfo) int {
		return s.compare(a.Name(), b.Name(), func() int {
			switch s.By {
			case Size:
				return cmp.Compare(a.Size(), b.Size())
			case Kind:
				return compareNames(kinds[a], kinds[b])
			case ModTime:
				return a.ModTime().Compare(b.ModTime())
			}
			return 0
		})
	})
}

// kindOf returns the kind of the given file, from its name.
func kindOf(info fs.FileInfo) string {
	if info.IsDir() {
		return "Folder"
	}
	fi := &fileinfo.FileInfo{Path: info.Name(), Name: info.Name()}
	fi.MimeFromFilename()
	return fi.Kind
}
