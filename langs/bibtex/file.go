// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bibtex

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// File maintains a record for an bibtex file
type File struct {

	// file name -- full path
	File string `desc:"file name -- full path"`

	// bibtex data loaded from file
	BibTex *BibTex `desc:"bibtex data loaded from file"`

	// mod time for loaded bibfile -- to detect updates
	Mod time.Time `desc:"mod time for loaded bibfile -- to detect updates"`
}

// FullPath returns full path to given bibtex file,
// looking on standard BIBINPUTS or TEXINPUTS env var paths if not found locally.
func FullPath(fname string) (string, error) {
	_, err := os.Stat(fname)
	path := fname
	nfErr := fmt.Errorf("bibtex file not found, even on BIBINPUTS or TEXINPUTS paths: %s", fname)
	if os.IsNotExist(err) {
		bin := os.Getenv("BIBINPUTS")
		if bin == "" {
			bin = os.Getenv("TEXINPUTS")
		}
		if bin == "" {
			return "", nfErr
		}
		pth := filepath.SplitList(bin)
		got := false
		for _, p := range pth {
			bf := filepath.Join(p, fname)
			_, err = os.Stat(bf)
			if err == nil {
				path = bf
				got = true
				break
			}
		}
		if !got {
			return "", nfErr
		}
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return path, err
	}
	return path, nil
}

// Open [re]opens the given filename, looking on standard BIBINPUTS or TEXINPUTS
// env var paths if not found locally.  If Mod >= mod timestamp on the file,
// and BibTex is already loaded, then nothing happens (already have it), but
// otherwise it parses the file and puts contents in BibTex field.
func (fl *File) Open(fname string) error {
	path := fname
	var err error
	if fl.File == "" {
		path, err = FullPath(fname)
		if err != nil {
			return err
		}
		fl.File = path
		fl.BibTex = nil
		fl.Mod = time.Time{}
		// fmt.Printf("first open file: %s path: %s\n", fname, fl.File)
	}
	st, err := os.Stat(fl.File)
	if err != nil {
		return err
	}
	if fl.BibTex != nil && !fl.Mod.Before(st.ModTime()) {
		// fmt.Printf("existing file: %v is fine: file mod: %v  last mod: %v\n", fl.File, st.ModTime(), fl.Mod)
		return nil
	}
	f, err := os.Open(fl.File)
	if err != nil {
		return err
	}
	defer f.Close()
	parsed, err := Parse(f)
	if err != nil {
		err = fmt.Errorf("Bibtex bibliography: %s not loaded due to error(s):\n%v", fl.File, err)
		return err
	}
	fl.BibTex = parsed
	fl.Mod = st.ModTime()
	// fmt.Printf("(re)loaded bibtex bibliography: %s\n", fl.File)
	return nil
}

//////////////////////////////////////////////////////////////////////
// Files

// Files is a map of bibtex files
type Files map[string]*File

// Open [re]opens the given filename, looking on standard BIBINPUTS or TEXINPUTS
// env var paths if not found locally.  If Mod >= mod timestamp on the file,
// and BibTex is already loaded, then nothing happens (already have it), but
// otherwise it parses the file and puts contents in BibTex field.
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
