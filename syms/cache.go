// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syms

import (
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goki/gi/gi"
)

// GoPiCacheDir returns the GoPi cache directory, and ensures that it exists
func GoPiCacheDir() (string, error) {
	ucdir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	cdir := filepath.Join(ucdir, "GoPi")
	err = os.MkdirAll(cdir, 0775)
	if err != nil {
		log.Printf("GoPiCacheDir: cache not available: %v\n", err)
	}
	return cdir, err
}

// GoRelPath returns the GOPATH or GOROOT relative path for given filename
func GoRelPath(filename string) (string, error) {
	absfn, err := filepath.Abs(filename)
	if err != nil {
		return absfn, err
	}
	relfn := absfn
	got := false
	for _, srcDir := range build.Default.SrcDirs() {
		if strings.HasPrefix(absfn, srcDir) {
			relfn = strings.TrimPrefix(absfn, srcDir)
			got = true
			break
		}
	}
	if got {
		return relfn, nil
	}
	homedir := gi.Prefs.User.HomeDir
	if strings.HasPrefix(absfn, homedir) {
		relfn = strings.TrimPrefix(absfn, homedir)
	}
	return relfn, nil
}

// CacheFilename returns the filename to use for cache file for given filename
func CacheFilename(filename string) (string, error) {
	cdir, err := GoPiCacheDir()
	if err != nil {
		return "", err
	}
	relfn, err := GoRelPath(filename)
	if err != nil {
		return "", err
	}
	path := relfn
	if filepath.Ext(path) != "" { // if it has an ext, it is not a dir..
		path, _ = filepath.Split(path)
	}
	path = filepath.Clean(path)
	if path[0] == filepath.Separator {
		path = path[1:]
	}
	path = strings.Replace(path, string(filepath.Separator), "_", -1)
	path = filepath.Join(cdir, path)
	return path, nil
}

// SaveSymCache saves cache of symbols starting with given symbol
// (typically a package, module, library), which is at given
// filename
func SaveSymCache(sy *Symbol, filename string) error {
	cfile, err := CacheFilename(filename)
	if err != nil {
		return err
	}
	return sy.SaveJSON(cfile)
}

// SaveSymDoc saves doc file of syms -- for double-checking contents etc
func SaveSymDoc(sy *Symbol, filename string) error {
	cfile, err := CacheFilename(filename)
	if err != nil {
		return err
	}
	cfile += ".doc"
	ofl, err := os.Create(cfile)
	if err != nil {
		return err
	}
	sy.WriteDoc(ofl, 0)
	return nil
}

// OpenSymCache opens cache of symbols into given symbol
// (typically a package, module, library), which is at given
// filename -- returns time stamp when cache was last saved
func OpenSymCache(filename string) (*Symbol, time.Time, error) {
	cfile, err := CacheFilename(filename)
	if err != nil {
		return nil, time.Time{}, err
	}
	info, err := os.Stat(cfile)
	if os.IsNotExist(err) {
		return nil, time.Time{}, err
	}
	sy := &Symbol{}
	err = sy.OpenJSON(cfile)
	return sy, info.ModTime(), err
}
