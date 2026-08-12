// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texcache

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/gzipx"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/text/shaped"
)

var (
	// cache is a cache of equations and a corresponding SVG-encoded path string
	cache map[string]*cached

	// mu guards all operations on cache
	mu sync.Mutex
)

// cached is one cache entry
type cached struct {
	// svg encoded path string
	Path string

	// font size in dots that it was encoded in
	// use scale transform to render in new target size.
	FontSizeDots float32

	// actual loaded, scaled PPath.
	pp ppath.Path

	// call DeleteUnused to remove any that haven't been accessed or added.
	used bool
}

// SetShapeMath sets the standard text shaping math function to use only
// cached math paths, not live generation of paths from TeX. This saves
// a lot of memory in the resulting executable, and is recommended for
// any math-heavy uses where all the equations can be cached in advance
// (e.g., the content system).
func SetShapeMath() {
	shaped.ShapeMath = func(expr string, fontSizeDots float32) (ppath.Path, error) {
		expr = strings.TrimSpace(expr)
		p := Get(expr, fontSizeDots)
		if p != nil {
			return p, nil
		}
		return nil, fmt.Errorf("texcache: no cached path for expression: %q", expr)
	}
}

// Get tries to get a cached path for given equation at given fontsize in dots.
// returns nil if not available.
func Get(expr string, fontSizeDots float32) ppath.Path {
	mu.Lock()
	defer mu.Unlock()

	cd, ok := cache[expr]
	if !ok {
		return nil
	}
	cd.used = true
	if cd.pp == nil { // lazy loading
		p, err := ppath.ParseSVGPath(cd.Path)
		if err != nil {
			fmt.Println("texcache parsing error -- shouldn't happen!", err)
			delete(cache, expr)
			return nil
		}
		cd.pp = p
	}
	if cd.FontSizeDots == fontSizeDots {
		return cd.pp
	}
	sc := fontSizeDots / cd.FontSizeDots
	np := cd.pp.Scale(sc, sc)
	cache[expr] = &cached{FontSizeDots: fontSizeDots, pp: np, used: true}
	return np
}

// Add adds a new entry
func Add(expr string, fontSizeDots float32, pp ppath.Path) {
	mu.Lock()
	defer mu.Unlock()

	if cache == nil {
		cache = make(map[string]*cached)
	}
	cache[expr] = &cached{FontSizeDots: fontSizeDots, pp: pp, used: true}
}

func setCache(c map[string]*cached) {
	if cache == nil {
		cache = c
		return
	}
	for k, v := range c {
		cache[k] = v
	}
}

// OpenFS opens saved cache entries from given file system (e.g., for embed).
// If the filename ends in .gz, it is unzipped for reading.
func OpenFS(fsys fs.FS, filename string) error {
	mu.Lock()
	defer mu.Unlock()

	var c map[string]*cached
	err := gzipx.OpenFS(fsys, filename, func(r io.Reader) error {
		return jsonx.Read(&c, r)
	})
	if err != nil {
		return errors.Log(err)
	}
	setCache(c)
	return nil
}

// Open opens saved cache entries from given file
// If the filename ends in .gz, it is unzipped for reading.
func Open(filename string) error {
	mu.Lock()
	defer mu.Unlock()

	var c map[string]*cached
	err := gzipx.Open(filename, func(r io.Reader) error {
		return jsonx.Read(&c, r)
	})
	if err != nil {
		return errors.Log(err)
	}
	setCache(c)
	return nil
}

// DeleteUnused deletes any cache entries that have not been
// accessed since the cache was loaded.
// Use this in generate functions that save all generated cache.
func DeleteUnused() {
	for k, v := range cache {
		if !v.used {
			delete(cache, k)
		}
	}
}

// SaveAs saves cache entries to given file
// If the filename ends in .gz, it is zipped.
// When generating, can call DeleteUnused to filter
// all unused items.
func SaveAs(filename string) error {
	mu.Lock()
	defer mu.Unlock()

	if cache == nil {
		return fmt.Errorf("texcache.SaveAs: no cache entries!")
	}
	for _, v := range cache {
		v.Path = v.pp.ToSVG()
	}

	err := gzipx.Save(filename, func(w io.Writer) error {
		return jsonx.Write(cache, w)
	})
	return errors.Log(err)
}
