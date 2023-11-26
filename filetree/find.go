// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"goki.dev/gi/v2/gi"
	"goki.dev/ki/v2"
	"goki.dev/pi/v2/filecat"
)

// FindDirNode finds directory node by given path.
// Must be a relative path already rooted at tree, or absolute path within tree.
func (fn *Node) FindDirNode(path string) (*Node, error) {
	rp := fn.RelPath(gi.FileName(path))
	if rp == "" {
		return nil, fmt.Errorf("FindDirNode: path: %s is not relative to this node's path: %s", path, fn.FPath)
	}
	if rp == "." {
		return fn, nil
	}
	dirs := filepath.SplitList(rp)
	dir := dirs[0]
	dni, err := fn.ChildByNameTry(dir, 0)
	if err != nil {
		return nil, err
	}
	dn := AsNode(dni)
	if len(dirs) == 1 {
		if dn.IsDir() {
			return dn, nil
		}
		return nil, fmt.Errorf("FindDirNode: item at path: %s is not a Directory", path)
	}
	return dn.FindDirNode(filepath.Join(dirs[1:]...))
}

// FindFile finds first node representing given file (false if not found) --
// looks for full path names that have the given string as their suffix, so
// you can include as much of the path (including whole thing) as is relevant
// to disambiguate.  See FilesMatching for a list of files that match a given
// string.
func (fn *Node) FindFile(fnm string) (*Node, bool) {
	if fnm == "" {
		return nil, false
	}
	fneff := fnm
	if len(fneff) > 2 && fneff[:2] == ".." { // relative path -- get rid of it and just look for relative part
		dirs := strings.Split(fneff, string(filepath.Separator))
		for i, dr := range dirs {
			if dr != ".." {
				fneff = filepath.Join(dirs[i:]...)
				break
			}
		}
	}

	if efn, err := fn.FRoot.ExtNodeByPath(fnm); err == nil {
		return efn, true
	}

	if strings.HasPrefix(fneff, string(fn.FPath)) { // full path
		ffn, err := fn.DirsTo(fneff)
		if err == nil {
			return ffn, true
		}
		return nil, false
	}

	var ffn *Node
	found := false
	fn.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		sfn := AsNode(wi)
		if sfn == nil {
			return ki.Continue
		}
		if strings.HasSuffix(string(sfn.FPath), fneff) {
			ffn = sfn
			found = true
			return ki.Break
		}
		return ki.Continue
	})
	return ffn, found
}

// FilesMatching returns list of all nodes whose file name contains given
// string (no regexp). ignoreCase transforms everything into lowercase
func (fn *Node) FilesMatching(match string, ignoreCase bool) []*Node {
	mls := make([]*Node, 0)
	if ignoreCase {
		match = strings.ToLower(match)
	}
	fn.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		sfn := AsNode(wi)
		if sfn == nil {
			return ki.Continue
		}
		if ignoreCase {
			nm := strings.ToLower(sfn.Nm)
			if strings.Contains(nm, match) {
				mls = append(mls, sfn)
			}
		} else {
			if strings.Contains(sfn.Nm, match) {
				mls = append(mls, sfn)
			}
		}
		return ki.Continue
	})
	return mls
}

// NodeNameCount is used to report counts of different string-based things
// in the file tree
type NodeNameCount struct {
	Name  string
	Count int
}

func NodeNameCountSort(ecs []NodeNameCount) {
	sort.Slice(ecs, func(i, j int) bool {
		return ecs[i].Count > ecs[j].Count
	})
}

// FileExtCounts returns a count of all the different file extensions, sorted
// from highest to lowest.
// If cat is != filecat.Unknown then it only uses files of that type
// (e.g., filecat.Code to find any code files)
func (fn *Node) FileExtCounts(cat filecat.Cat) []NodeNameCount {
	cmap := make(map[string]int, 20)
	fn.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		sfn := AsNode(wi)
		if sfn == nil {
			return ki.Continue
		}
		if cat != filecat.Unknown {
			if sfn.Info.Cat != cat {
				return ki.Continue
			}
		}
		ext := strings.ToLower(filepath.Ext(sfn.Nm))
		if ec, has := cmap[ext]; has {
			cmap[ext] = ec + 1
		} else {
			cmap[ext] = 1
		}
		return ki.Continue
	})
	ecs := make([]NodeNameCount, len(cmap))
	idx := 0
	for key, val := range cmap {
		ecs[idx] = NodeNameCount{Name: key, Count: val}
		idx++
	}
	NodeNameCountSort(ecs)
	return ecs
}

// LatestFileMod returns the most recent mod time of files in the tree.
// If cat is != filecat.Unknown then it only uses files of that type
// (e.g., filecat.Code to find any code files)
func (fn *Node) LatestFileMod(cat filecat.Cat) time.Time {
	tmod := time.Time{}
	fn.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		sfn := AsNode(wi)
		if sfn == nil {
			return ki.Continue
		}
		if cat != filecat.Unknown {
			if sfn.Info.Cat != cat {
				return ki.Continue
			}
		}
		ft := (time.Time)(sfn.Info.ModTime)
		if ft.After(tmod) {
			tmod = ft
		}
		return ki.Continue
	})
	return tmod
}
