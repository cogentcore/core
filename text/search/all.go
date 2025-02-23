// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filesearch

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
)

// SearchAll returns list of all files starting at given file path,
// of given language(s) that contain the given string,
// sorted in descending order by number of occurrences.
// ignoreCase transforms everything into lowercase.
// exclude is a list of filenames to exclude.
func SearchAll(start string, find string, ignoreCase, regExp bool, langs []fileinfo.Known, exclude ...string) []Results {
	fb := []byte(find)
	fsz := len(find)
	if fsz == 0 {
		return nil
	}
	var re *regexp.Regexp
	var err error
	if regExp {
		re, err = regexp.Compile(find)
		if err != nil {
			log.Println(err)
			return nil
		}
	}
	mls := make([]Results, 0)
	spath := string(start.Filepath) // note: is already Abs
	filepath.Walk(spath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == ".git" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if int(info.Size()) > core.SystemSettings.BigFileSize {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".code") { // exclude self
			return nil
		}
		if fileinfo.IsGeneratedFile(path) {
			return nil
		}
		if len(langs) > 0 {
			mtyp, _, err := fileinfo.MimeFromFile(path)
			if err != nil {
				return nil
			}
			known := fileinfo.MimeKnown(mtyp)
			if !fileinfo.IsMatchList(langs, known) {
				return nil
			}
		}
		ofn := openPath(path)
		var cnt int
		var matches []lines.Match
		if ofn != nil && ofn.Buffer != nil {
			if regExp {
				cnt, matches = ofn.Buffer.SearchRegexp(re)
			} else {
				cnt, matches = ofn.Buffer.Search(fb, ignoreCase, false)
			}
		} else {
			if regExp {
				cnt, matches = lines.SearchFileRegexp(path, re)
			} else {
				cnt, matches = lines.SearchFile(path, fb, ignoreCase)
			}
		}
		if cnt > 0 {
			if ofn != nil {
				mls = append(mls, Results{ofn, cnt, matches})
			} else {
				sfn, found := start.FindFile(path)
				if found {
					mls = append(mls, Results{sfn, cnt, matches})
				} else {
					fmt.Println("file not found in FindFile:", path)
				}
			}
		}
		return nil
	})
	sort.Slice(mls, func(i, j int) bool {
		return mls[i].Count > mls[j].Count
	})
	return mls
}
