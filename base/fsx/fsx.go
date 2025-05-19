// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fsx provides various utility functions for dealing with filesystems.
package fsx

import (
	"errors"
	"fmt"
	"go/build"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Filename is used to open a file picker dialog when used as an argument
// type in a function, or as a field value.
type Filename string

// GoSrcDir tries to locate dir in GOPATH/src/ or GOROOT/src/pkg/ and returns its
// full path. GOPATH may contain a list of paths. From Robin Elkind github.com/mewkiz/pkg.
func GoSrcDir(dir string) (absDir string, err error) {
	for _, srcDir := range build.Default.SrcDirs() {
		absDir = filepath.Join(srcDir, dir)
		finfo, err := os.Stat(absDir)
		if err == nil && finfo.IsDir() {
			return absDir, nil
		}
	}
	return "", fmt.Errorf("fsx.GoSrcDir: unable to locate directory (%q) in GOPATH/src/ (%q) or GOROOT/src/pkg/ (%q)", dir, os.Getenv("GOPATH"), os.Getenv("GOROOT"))
}

// Files returns all the DirEntry's for files with given extension(s) in directory
// in sorted order (if extensions are empty then all files are returned).
// In case of error, returns nil.
func Files(path string, extensions ...string) []fs.DirEntry {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	if len(extensions) == 0 {
		return files
	}
	sz := len(files)
	if sz == 0 {
		return nil
	}
	for i := sz - 1; i >= 0; i-- {
		fn := files[i]
		ext := filepath.Ext(fn.Name())
		keep := false
		for _, ex := range extensions {
			if strings.EqualFold(ext, ex) {
				keep = true
				break
			}
		}
		if !keep {
			files = append(files[:i], files[i+1:]...)
		}
	}
	return files
}

// Filenames returns all the file names with given extension(s) in directory
// in sorted order (if extensions is empty then all files are returned)
func Filenames(path string, extensions ...string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	files, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil
	}
	if len(extensions) == 0 {
		sort.StringSlice(files).Sort()
		return files
	}
	sz := len(files)
	if sz == 0 {
		return nil
	}
	for i := sz - 1; i >= 0; i-- {
		fn := files[i]
		ext := filepath.Ext(fn)
		keep := false
		for _, ex := range extensions {
			if strings.EqualFold(ext, ex) {
				keep = true
				break
			}
		}
		if !keep {
			files = append(files[:i], files[i+1:]...)
		}
	}
	sort.StringSlice(files).Sort()
	return files
}

// Dirs returns a slice of all the directories within a given directory
func Dirs(path string) []string {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	var fnms []string
	for _, fi := range files {
		if fi.IsDir() {
			fnms = append(fnms, fi.Name())
		}
	}
	return fnms
}

// LatestMod returns the latest (most recent) modification time for any of the
// files in the directory (optionally filtered by extension(s) if exts != nil)
// if no files or error, returns zero time value
func LatestMod(path string, exts ...string) time.Time {
	tm := time.Time{}
	files := Files(path, exts...)
	if len(files) == 0 {
		return tm
	}
	for _, de := range files {
		fi, err := de.Info()
		if err == nil {
			if fi.ModTime().After(tm) {
				tm = fi.ModTime()
			}
		}
	}
	return tm
}

// HasFile returns true if given directory has given file (exact match)
func HasFile(path, file string) bool {
	files, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, fn := range files {
		if fn.Name() == file {
			return true
		}
	}
	return false
}

// FindFilesOnPaths attempts to locate given file(s) on given list of paths,
// returning the full Abs path to each file found (nil if none)
func FindFilesOnPaths(paths []string, files ...string) []string {
	var res []string
	for _, path := range paths {
		for _, fn := range files {
			fp := filepath.Join(path, fn)
			ok, _ := FileExists(fp)
			if ok {
				res = append(res, fp)
			}
		}
	}
	return res
}

// FileExists checks whether given file exists, returning true if so,
// false if not, and error if there is an error in accessing the file.
func FileExists(filePath string) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		return !fileInfo.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// DirAndFile returns the final dir and file name.
func DirAndFile(file string) string {
	dir, fnm := filepath.Split(file)
	return filepath.Join(filepath.Base(dir), fnm)
}

// RelativeFilePath returns the file name relative to given root file path, if it is
// under that root; otherwise it returns the final dir and file name.
func RelativeFilePath(file, root string) string {
	rp, err := filepath.Rel(root, file)
	if err == nil && !strings.HasPrefix(rp, "..") {
		return rp
	}
	return DirAndFile(file)
}

// ExtSplit returns the split between the extension and name before
// the extension, for the given file name.  Any path elements in the
// file name are preserved; pass [filepath.Base](file) to extract only the
// last element of the file path if that is what is desired.
func ExtSplit(file string) (base, ext string) {
	ext = filepath.Ext(file)
	base = strings.TrimSuffix(file, ext)
	return
}

// here's all the discussion about why CopyFile is not in std lib:
// https://old.reddit.com/r/golang/comments/3lfqoh/why_golang_does_not_provide_a_copy_file_func/
// https://github.com/golang/go/issues/8868

// CopyFile copies the contents from src to dst atomically.
// If dst does not exist, CopyFile creates it with permissions perm.
// If the copy fails, CopyFile aborts and dst is preserved.
func CopyFile(dst, src string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), "")
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, in)
	if err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err = tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err = os.Chmod(tmp.Name(), perm); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), dst)
}
