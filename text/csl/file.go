// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

import (
	"io/fs"
	"os"
	"time"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/text/parse/languages/bibtex"
)

// Open opens CSL data items from a .json formatted CSL file.
func Open(filename string) ([]Item, error) {
	var its []Item
	err := jsonx.Open(&its, filename)
	return its, err
}

// OpenFS opens CSL data items from a .json formatted CSL file from given
// filesystem.
func OpenFS(fsys fs.FS, filename string) ([]Item, error) {
	var its []Item
	err := jsonx.OpenFS(&its, fsys, filename)
	return its, err
}

// SaveItems saves items to given filename.
func SaveItems(items []Item, filename string) error {
	return jsonx.Save(items, filename)
}

// SaveKeyList saves items to given filename.
func SaveKeyList(kl *KeyList, filename string) error {
	return jsonx.Save(kl.Values, filename)
}

////////  File

// File maintains a record for a CSL file.
type File struct {

	// File name, full path.
	File string

	// Items from the file, as a KeyList for easy citation lookup.
	Items *KeyList

	// mod time for loaded file, to detect updates.
	Mod time.Time
}

// Open [re]opens the given filename, looking on standard BIBINPUTS or TEXINPUTS
// env var paths if not found locally. If Mod >= mod timestamp on the file,
// and is already loaded, then nothing happens (already have it), but
// otherwise it parses the file and puts contents in Items.
func (fl *File) Open(fname string) error {
	path := fname
	var err error
	if fl.File == "" {
		path, err = bibtex.FullPath(fname)
		if err != nil {
			return err
		}
		fl.File = path
		fl.Items = nil
		fl.Mod = time.Time{}
		// fmt.Printf("first open file: %s path: %s\n", fname, fl.File)
	}
	st, err := os.Stat(fl.File)
	if err != nil {
		return err
	}
	if fl.Items != nil && !fl.Mod.Before(st.ModTime()) {
		// fmt.Printf("existing file: %v is fine: file mod: %v  last mod: %v\n", fl.File, st.ModTime(), fl.Mod)
		return nil
	}
	its, err := Open(fl.File)
	if err != nil {
		return err
	}
	fl.Items = NewKeyList(its)
	fl.Mod = st.ModTime()
	// fmt.Printf("(re)loaded bibtex bibliography: %s\n", fl.File)
	return nil
}

//////// Files

// Files is a map of CSL items keyed by file name.
type Files map[string]*File

// Open [re]opens the given filename, looking on standard BIBINPUTS or TEXINPUTS
// env var paths if not found locally.  If Mod >= mod timestamp on the file,
// and Items is already loaded, then nothing happens (already have it), but
// otherwise it parses the file and puts contents in Items field.
func (fl *Files) Open(fname string) (*File, error) {
	if *fl == nil {
		*fl = make(Files)
	}
	fr, has := (*fl)[fname]
	if has {
		err := fr.Open(fname)
		return fr, err
	}
	fr = &File{}
	err := fr.Open(fname)
	if err != nil {
		return nil, err
	}
	(*fl)[fname] = fr
	return fr, nil
}
