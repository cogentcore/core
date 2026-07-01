// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gzipx provides a convenience wrapper for gzip encoding
// for reader and writer.
package gzipx

import (
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Open opens given file and calls the given function on it,
// first detecting whether the filename has a .gz extension,
// and adding an gzip reader if so.
func Open(filename string, fun func(gzr io.Reader) error) error {
	fp, err := os.Open(filename)
	defer fp.Close()
	if err != nil {
		return err
	}
	ext := filepath.Ext(filename)
	if ext == ".gz" {
		return Read(fp, fun)
	}
	return fun(fp)
}

// OpenFS opens given file from filesystem and calls the given function on it,
// first detecting whether the filename has a .gz extension,
// and adding a gzip reader if so.
func OpenFS(fsys fs.FS, filename string, fun func(gzr io.Reader) error) error {
	fp, err := fsys.Open(filename)
	defer fp.Close()
	if err != nil {
		return err
	}
	ext := filepath.Ext(filename)
	if ext == ".gz" {
		return Read(fp, fun)
	}
	return fun(fp)
}

// Read adds a gzip reader to the given reader,
// and calls the given function with the unzip reader,
// returning the error from that function.
func Read(r io.Reader, fun func(gzr io.Reader) error) error {
	gzr, err := gzip.NewReader(r)
	defer gzr.Close()
	if err != nil {
		return err
	}
	return fun(gzr)
}

// Save opens given file and calls the given function on it,
// first detecting whether the filename has a .gz extension,
// and adding a gzip writer if so.
func Save(filename string, fun func(gzw io.Writer) error) error {
	fp, err := os.Create(filename)
	defer fp.Close()
	if err != nil {
		return err
	}
	ext := filepath.Ext(filename)
	if ext == ".gz" {
		return Write(fp, fun)
	}
	return fun(fp)
}

// Write adds a gzip writer to the given writer,
// and calls the given function with the gzip writer,
// returning the error from that function.
func Write(w io.Writer, fun func(gzw io.Writer) error) error {
	gzw := gzip.NewWriter(w)
	defer gzw.Close()
	return fun(gzw)
}
