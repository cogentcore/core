// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filesearch

import (
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/tree"
)

// SearchOpen returns list of all files starting at given file path, of
// language(s) that contain the given string, sorted in descending order by number
// of occurrences; ignoreCase transforms everything into lowercase.
// exclude is a list of filenames to exclude.
func Search(start string, find string, ignoreCase, regExp bool, loc Locations, langs []fileinfo.Known, exclude ...string) []Results {
	fb := []byte(find)
	fsz := len(find)
	if fsz == 0 {
		return nil
	}
	if loc == All {
		return findAll(start, find, ignoreCase, regExp, langs)
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
	start.WalkDown(func(k tree.Node) bool {
		sfn := AsNode(k)
		if sfn == nil {
			return tree.Continue
		}
		if sfn.IsDir() && !sfn.isOpen() {
			// fmt.Printf("dir: %v closed\n", sfn.FPath)
			return tree.Break // don't go down into closed directories!
		}
		if sfn.IsDir() || sfn.IsExec() || sfn.Info.Kind == "octet-stream" || sfn.isAutoSave() || sfn.Info.Generated {
			// fmt.Printf("dir: %v opened\n", sfn.Nm)
			return tree.Continue
		}
		if int(sfn.Info.Size) > core.SystemSettings.BigFileSize {
			return tree.Continue
		}
		if strings.HasSuffix(sfn.Name, ".code") { // exclude self
			return tree.Continue
		}
		if !fileinfo.IsMatchList(langs, sfn.Info.Known) {
			return tree.Continue
		}
		if loc == Dir {
			cdir, _ := filepath.Split(string(sfn.Filepath))
			if activeDir != cdir {
				return tree.Continue
			}
		} else if loc == NotTop {
			// if level == 1 { // todo
			// 	return tree.Continue
			// }
		}
		var cnt int
		var matches []textpos.Match
		if sfn.isOpen() && sfn.Lines != nil {
			if regExp {
				cnt, matches = sfn.Lines.SearchRegexp(re)
			} else {
				cnt, matches = sfn.Lines.Search(fb, ignoreCase, false)
			}
		} else {
			if regExp {
				cnt, matches = lines.SearchFileRegexp(string(sfn.Filepath), re)
			} else {
				cnt, matches = lines.SearchFile(string(sfn.Filepath), fb, ignoreCase)
			}
		}
		if cnt > 0 {
			mls = append(mls, Results{sfn, cnt, matches})
		}
		return tree.Continue
	})
	sort.Slice(mls, func(i, j int) bool {
		return mls[i].Count > mls[j].Count
	})
	return mls
}
