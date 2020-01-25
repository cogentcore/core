// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dirs provides various utility functions in dealing with directories
// such as a list of all the files with a given (set of) extensions and
// finding paths within the Go source directory (GOPATH, etc)
package dirs

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GoSrcDir tries to locate dir in GOPATH/src/ or GOROOT/src/pkg/ and returns its
// full path. GOPATH may contain a list of paths.  From Robin Elkind github.com/mewkiz/pkg
func GoSrcDir(dir string) (absDir string, err error) {
	for _, srcDir := range build.Default.SrcDirs() {
		absDir = filepath.Join(srcDir, dir)
		finfo, err := os.Stat(absDir)
		if err == nil && finfo.IsDir() {
			return absDir, nil
		}
	}
	/* this is probably redundant and not needed -- and UserHomeDir is only in 1.12
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	absDir = filepath.Join(filepath.Join(filepath.Join(home, "go"), "src"), dir)
	finfo, err := os.Stat(absDir)
	if err == nil && finfo.IsDir() {
		return absDir, nil
	}
	*/
	return "", fmt.Errorf("kit.GoSrcDir: unable to locate directory (%q) in GOPATH/src/ (%q) or GOROOT/src/pkg/ (%q)", dir, os.Getenv("GOPATH"), os.Getenv("GOROOT"))
}

// ExtFiles returns all the FileInfo's for files with given extension(s) in directory
// in sorted order (if exts is empty then all files are returned).
// In case of error, returns nil.
func ExtFiles(path string, exts []string) []os.FileInfo {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil
	}
	if len(exts) == 0 {
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
		for _, ex := range exts {
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

// ExtFileNames returns all the file names with given extension(s) in directory
// in sorted order (if exts is empty then all files are returned)
func ExtFileNames(path string, exts []string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	files, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil
	}
	if len(exts) == 0 {
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
		for _, ex := range exts {
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
	files, err := ioutil.ReadDir(path)
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
// files in the directory (optinally filtered by extension(s) if exts != nil)
// if no files or error, returns zero time value
func LatestMod(path string, exts []string) time.Time {
	tm := time.Time{}
	files := ExtFiles(path, exts)
	if len(files) == 0 {
		return tm
	}
	for _, fi := range files {
		if fi.ModTime().After(tm) {
			tm = fi.ModTime()
		}
	}
	return tm
}

// AllFiles returns a slice of all the files, recursively, within a given directory
func AllFiles(path string) ([]string, error) {
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

// HasFile returns true if given directory has given file (exact match)
func HasFile(path, file string) bool {
	files, err := ioutil.ReadDir(path)
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
