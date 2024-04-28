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

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/profile"
	"cogentcore.org/core/parse"
	"cogentcore.org/core/parse/lexer"
)

func init() {
	parse.LangSupport.OpenStandard()
}

func TestParse(t *testing.T) {
	// t.Skip("todo: reenable soon")
	lp, _ := parse.LangSupport.Properties(fileinfo.Go)
	pr := lp.Lang.Parser()
	pr.ReportErrs = true

	fs := parse.NewFileStates(filepath.Join("..", "..", "..", "views", "treeview.go"), "", fileinfo.Go)
	txt, err := lexer.OpenFileBytes(fs.Filename) // and other stuff
	if err != nil {
		t.Error(err)
	}

	profile.Profiling = true
	stt := time.Now()
	lp.Lang.ParseFile(fs, txt)
	prdur := time.Since(stt)
	fmt.Printf("core parse: %v\n", prdur)

	profile.Report(time.Millisecond)
	profile.Profiling = false
}

// Note: couldn't get benchmark to do anything reasonable on this one, so just
// using plain test on single iter

func TestGoParse(t *testing.T) {
	// t.Skip("todo: reenable soon")
	stt := time.Now()
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, filepath.Join("..", "..", "..", "views", "treeview.go"), nil, parser.ParseComments)
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
