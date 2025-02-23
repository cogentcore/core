// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package search

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textpos"
)

// excludeFile does the exclude match against either file name or file path,
// removes any problematic exclude expressions from the list.
func excludeFile(exclude *[]string, fname, fpath string) (bool, error) {
	var errs []error
	for ei, ex := range *exclude {
		exp, err := filepath.Match(ex, fpath)
		if err != nil {
			errs = append(errs, err)
			*exclude = slices.Delete(*exclude, ei, ei+1)
		}
		if exp {
			return true, errors.Join(errs...)
		}
		exf, _ := filepath.Match(ex, fname)
		if exf {
			return true, errors.Join(errs...)
		}
	}
	return false, errors.Join(errs...)
}

// langCheck checks if file matches list of target languages: true if
// matches (or no langs)
func langCheck(fi *fileinfo.FileInfo, langs []fileinfo.Known) bool {
	if len(langs) == 0 {
		return true
	}
	if fileinfo.IsMatchList(langs, fi.Known) {
		return true
	}
	return false
}

// Paths returns list of all files in given list of paths (only: no subdirs),
// of language(s) that contain the given string, sorted in descending order
// by number of occurrences. Paths can be relative to current working directory.
// Automatically skips generated files.
//   - ignoreCase transforms everything into lowercase.
//   - regExp uses the go regexp syntax for the find string.
//   - exclude is a list of filenames to exclude: can use standard Glob patterns.
func Paths(paths []string, find string, ignoreCase, regExp bool, langs []fileinfo.Known, exclude ...string) ([]Results, error) {
	fsz := len(find)
	if fsz == 0 {
		return nil, nil
	}
	fb := []byte(find)
	var re *regexp.Regexp
	var err error
	if regExp {
		re, err = regexp.Compile(find)
		if err != nil {
			return nil, err
		}
	}
	mls := make([]Results, 0)
	var errs []error
	for _, path := range paths {
		files, err := os.ReadDir(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, de := range files {
			if de.IsDir() {
				continue
			}
			fname := de.Name()
			fpath := filepath.Join(path, fname)
			skip, err := excludeFile(&exclude, fname, fpath)
			if err != nil {
				errs = append(errs, err)
			}
			if skip {
				continue
			}
			fi, err := fileinfo.NewFileInfo(fpath)
			if err != nil {
				errs = append(errs, err)
			}
			if int(fi.Size) > core.SystemSettings.BigFileSize {
				continue
			}
			if fi.Generated {
				continue
			}
			if !langCheck(fi, langs) {
				continue
			}
			var cnt int
			var matches []textpos.Match
			if regExp {
				cnt, matches = FileRegexp(fpath, re)
			} else {
				cnt, matches = File(fpath, fb, ignoreCase)
			}
			if cnt > 0 {
				fpabs, err := filepath.Abs(fpath)
				if err != nil {
					errs = append(errs, err)
				}
				mls = append(mls, Results{fpabs, cnt, matches})
			}
		}
	}
	sort.Slice(mls, func(i, j int) bool {
		return mls[i].Count > mls[j].Count
	})
	return mls, errors.Join(errs...)
}
