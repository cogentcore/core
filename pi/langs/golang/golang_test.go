// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/fi"
	"cogentcore.org/core/pi"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/prof"
)

func init() {
	pi.LangSupport.OpenStd()
}

func TestParse(t *testing.T) {
	// t.Skip("todo: reenable soon")
	lp, _ := pi.LangSupport.Props(fi.Go)
	pr := lp.Lang.Parser()
	pr.ReportErrs = true

	fs := pi.NewFileStates(filepath.Join("testdata", "treeview.go"), "", fi.Go)
	txt, err := lex.OpenFileBytes(fs.Filename) // and other stuff
	if err != nil {
		t.Error(err)
	}

	prof.Profiling = true
	stt := time.Now()
	lp.Lang.ParseFile(fs, txt)
	prdur := time.Since(stt)
	fmt.Printf("pi parse: %v\n", prdur)

	prof.Report(time.Millisecond)
	prof.Profiling = false
}

// Note: couldn't get benchmark to do anything reasonable on this one, so just
// using plain test on single iter

func TestGoParse(t *testing.T) {
	// t.Skip("todo: reenable soon")
	stt := time.Now()
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, filepath.Join("testdata", "treeview.go"), nil, parser.ParseComments)
	if err != nil {
		t.Error(err)
	}
	prdur := time.Since(stt)
	fmt.Printf("go parse: %v\n", prdur)

	// fmt.Println("Functions:")
	// for _, f := range node.Decls {
	// 	fn, ok := f.(*ast.FuncDecl)
	// 	if !ok {
	// 		continue
	// 	}
	// 	fmt.Println(fn.Name.Name)
	// }
}
