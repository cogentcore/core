// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

import (
	"io/fs"
	"os"
	"slices"
	"strings"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/stringsx"
	"golang.org/x/exp/maps"
)

type Dict map[string]struct{}

func (d Dict) Add(word string) {
	d[word] = struct{}{}
}

func (d Dict) Exists(word string) bool {
	_, ex := d[word]
	return ex
}

// List returns a list (slice) of words in dictionary
// in alpha-sorted order
func (d Dict) List() []string {
	wl := maps.Keys(d)
	slices.Sort(wl)
	return wl
}

// Save saves a dictionary list of words
// to a simple one-word-per-line list, in alpha order
func (d Dict) Save(fname string) error {
	wl := d.List()
	ws := strings.Join(wl, "\n")
	return os.WriteFile(fname, []byte(ws), 0666)
}

// NewDictFromList makes a new dictionary from given list
// (slice) of words
func NewDictFromList(wl []string) Dict {
	d := make(Dict, len(wl))
	for _, w := range wl {
		d.Add(w)
	}
	return d
}

// OpenDict opens a dictionary list of words
// from a simple one-word-per-line list
func OpenDict(fname string) (Dict, error) {
	dfs, fnm, err := fsx.DirFS(fname)
	if err != nil {
		return nil, err
	}
	return OpenDictFS(dfs, fnm)
}

// OpenDictFS opens a dictionary list of words
// from a simple one-word-per-line list, from given filesystem
func OpenDictFS(fsys fs.FS, filename string) (Dict, error) {
	f, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return nil, err
	}
	wl := stringsx.SplitLines(string(f))
	d := NewDictFromList(wl)
	return d, nil
}
