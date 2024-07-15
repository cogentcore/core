// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

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
	"cogentcore.org/core/texteditor/textbuf"
	"cogentcore.org/core/tree"
)

// FindLocation corresponds to the search scope
type FindLocation int32 //enums:enum -trim-prefix FindLocation

const (
	// FindOpen finds in all open folders in the left file browser
	FindLocationOpen FindLocation = iota

	// FindLocationAll finds in all directories under the root path. can be slow for large file trees
	FindLocationAll

	// FindLocationFile only finds in the current active file
	FindLocationFile

	// FindLocationDir only finds in the directory of the current active file
	FindLocationDir

	// FindLocationNotTop finds in all open folders *except* the top-level folder
	FindLocationNotTop
)

// SearchResults is used to report search results
type SearchResults struct {
	Node    *Node
	Count   int
	Matches []textbuf.Match
}

// Search returns list of all nodes starting at given node of given
// language(s) that contain the given string, sorted in descending order by number
// of occurrences; ignoreCase transforms everything into lowercase
func Search(start *Node, find string, ignoreCase, regExp bool, loc FindLocation, activeDir string, langs []fileinfo.Known, openPath func(path string) *Node) []SearchResults {
	fb := []byte(find)
	fsz := len(find)
	if fsz == 0 {
		return nil
	}
	if loc == FindLocationAll {
		return findAll(start, find, ignoreCase, regExp, langs, openPath)
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
	mls := make([]SearchResults, 0)
	start.WalkDown(func(k tree.Node) bool {
		sfn := AsNode(k)
		if sfn == nil {
			return tree.Continue
		}
		if sfn.IsDir() && !sfn.isOpen() {
			// fmt.Printf("dir: %v closed\n", sfn.FPath)
			return tree.Break // don't go down into closed directories!
		}
		if sfn.IsDir() || sfn.IsExec() || sfn.Info.Kind == "octet-stream" || sfn.isAutoSave() {
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
		if loc == FindLocationDir {
			cdir, _ := filepath.Split(string(sfn.Filepath))
			if activeDir != cdir {
				return tree.Continue
			}
		} else if loc == FindLocationNotTop {
			// if level == 1 { // todo
			// 	return tree.Continue
			// }
		}
		var cnt int
		var matches []textbuf.Match
		if sfn.isOpen() && sfn.Buffer != nil {
			if regExp {
				cnt, matches = sfn.Buffer.SearchRegexp(re)
			} else {
				cnt, matches = sfn.Buffer.Search(fb, ignoreCase, false)
			}
		} else {
			if regExp {
				cnt, matches = textbuf.SearchFileRegexp(string(sfn.Filepath), re)
			} else {
				cnt, matches = textbuf.SearchFile(string(sfn.Filepath), fb, ignoreCase)
			}
		}
		if cnt > 0 {
			mls = append(mls, SearchResults{sfn, cnt, matches})
		}
		return tree.Continue
	})
	sort.Slice(mls, func(i, j int) bool {
		return mls[i].Count > mls[j].Count
	})
	return mls
}

// findAll returns list of all files (regardless of what is currently open)
// starting at given node of given language(s) that contain the given string,
// sorted in descending order by number of occurrences. ignoreCase transforms
// everything into lowercase.
func findAll(start *Node, find string, ignoreCase, regExp bool, langs []fileinfo.Known, openPath func(path string) *Node) []SearchResults {
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
	mls := make([]SearchResults, 0)
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
		var matches []textbuf.Match
		if ofn != nil && ofn.Buffer != nil {
			if regExp {
				cnt, matches = ofn.Buffer.SearchRegexp(re)
			} else {
				cnt, matches = ofn.Buffer.Search(fb, ignoreCase, false)
			}
		} else {
			if regExp {
				cnt, matches = textbuf.SearchFileRegexp(path, re)
			} else {
				cnt, matches = textbuf.SearchFile(path, fb, ignoreCase)
			}
		}
		if cnt > 0 {
			if ofn != nil {
				mls = append(mls, SearchResults{ofn, cnt, matches})
			} else {
				sfn, found := start.FindFile(path)
				if found {
					mls = append(mls, SearchResults{sfn, cnt, matches})
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
