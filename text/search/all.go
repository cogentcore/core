// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package search

import (
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textpos"
)

// All returns list of all files under given root path, in all subdirs,
// of given language(s) that contain the given string, sorted in
// descending order by number of occurrences.
//   - ignoreCase transforms everything into lowercase.
//   - regExp uses the go regexp syntax for the find string.
//   - exclude is a list of filenames to exclude.
func All(root string, find string, ignoreCase, regExp bool, langs []fileinfo.Known, exclude ...string) ([]Results, error) {
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
	filepath.Walk(root, func(fpath string, info fs.FileInfo, err error) error {
		if err != nil {
			errs = append(errs, err)
			return err
		}
		if info.IsDir() {
			return nil
		}
		if int(info.Size()) > core.SystemSettings.BigFileSize {
			return nil
		}
		fname := info.Name()
		skip, err := excludeFile(&exclude, fname, fpath)
		if err != nil {
			errs = append(errs, err)
		}
		if skip {
			return nil
		}
		fi, err := fileinfo.NewFileInfo(fpath)
		if err != nil {
			errs = append(errs, err)
		}
		if fi.Generated {
			return nil
		}
		if !langCheck(fi, langs) {
			return nil
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
		return nil
	})
	sort.Slice(mls, func(i, j int) bool {
		return mls[i].Count > mls[j].Count
	})
	return mls, errors.Join(errs...)
}
