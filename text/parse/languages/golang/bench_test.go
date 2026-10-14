// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/lexer"
)

// benchFile is the Go source file parsed by the benchmarks.
var benchFile = filepath.Join("..", "..", "..", "..", "core", "tree.go")

// benchParser returns the Go parser and the source of [benchFile].
func benchParser(b *testing.B) (*parse.Parser, []byte) {
	lp, _ := parse.LanguageSupport.Properties(fileinfo.Go)
	txt, err := lexer.OpenFileBytes(benchFile)
	if err != nil {
		b.Fatal(err)
	}
	return lp.Lang.Parser(), txt
}

// benchFileState returns a new file state for [benchFile] with the given
// source, set up as the text editor does it.
func benchFileState(txt []byte) *parse.FileState {
	fss := parse.NewFileStates(benchFile, "", fileinfo.Go)
	fs := fss.StartProc(txt)
	fss.EndProc()
	return fs
}

// BenchmarkLexFile measures lexing a Go file, which is what syntax
// highlighting uses for each line.
func BenchmarkLexFile(b *testing.B) {
	pr, txt := benchParser(b)
	for b.Loop() {
		pr.LexAll(benchFileState(txt))
	}
}

// BenchmarkParseFile measures lexing and parsing a Go file, which is what
// the text editor does when it opens a Go file.
func BenchmarkParseFile(b *testing.B) {
	pr, txt := benchParser(b)
	for b.Loop() {
		fs := benchFileState(txt)
		pr.LexAll(fs)
		pr.ParseAll(fs)
	}
}

// BenchmarkReparseFile measures lexing and parsing a Go file again with the
// same file states, which is what the text editor does after each edit.
func BenchmarkReparseFile(b *testing.B) {
	pr, txt := benchParser(b)
	fss := parse.NewFileStates(benchFile, "", fileinfo.Go)
	for b.Loop() {
		fs := fss.StartProc(txt)
		pr.LexAll(fs)
		pr.ParseAll(fs)
		fss.EndProc()
	}
}
