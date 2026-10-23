// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lexers provides the chroma lexers for syntax highlighting one
// language at a time, so that an app links only the definitions of the
// languages that it highlights, instead of those of all of the hundreds of
// languages in github.com/alecthomas/chroma/v3/lexers. Each function adds
// the lexer for a language to a registry, as in:
//
//	highlighting.AddLexers(lexers.Bash, lexers.TOML)
//
// The lexers are copies of the chroma lexers defined in XML, and Go. Each
// function also adds the lexers that its lexer uses, such as the CSS and
// JavaScript lexers for HTML.
// Copyright (C) 2017 Alec Thomas, under an MIT License (see COPYING).
package lexers

//go:generate go run gen.go

import (
	"io/fs"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v3"
)

// register adds the lexer defined by the XML file with the given name and
// contents to the registry. Its rules are loaded when it is first used.
func register(reg *chroma.LexerRegistry, file, xml string) {
	reg.Register(chroma.MustNewXMLLexer(xmlFS{file, xml}, file))
}

// xmlFS is a file system with the XML definition of a lexer as its only file.
// The definitions are embedded as strings rather than embed.FS variables,
// because the bundler of the web leaves out unused strings, but not unused
// embed.FS variables.
type xmlFS struct {
	name, data string
}

func (x xmlFS) Open(name string) (fs.File, error) {
	if name != x.name {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &xmlFile{Reader: strings.NewReader(x.data), name: name}, nil
}

// xmlFile is an open file of an [xmlFS], which is also its [fs.FileInfo].
type xmlFile struct {
	*strings.Reader
	name string
}

func (f *xmlFile) Stat() (fs.FileInfo, error) { return f, nil }
func (f *xmlFile) Close() error               { return nil }
func (f *xmlFile) Name() string               { return f.name }
func (f *xmlFile) Mode() fs.FileMode          { return 0444 }
func (f *xmlFile) ModTime() time.Time         { return time.Time{} }
func (f *xmlFile) IsDir() bool                { return false }
func (f *xmlFile) Sys() any                   { return nil }
