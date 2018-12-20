// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"fmt"
	"go/parser"
	"go/token"
	"testing"
	"time"

	"github.com/goki/gi/filecat"
	"github.com/goki/prof"
)

func init() {
	LangSupport.OpenStd()
}

func TestParse(t *testing.T) {
	lp, _ := LangSupport.Props(filecat.Go)
	pr := lp.Lang.Parser()
	pr.ReportErrs = true

	fs := NewFileState()
	err := fs.Src.OpenFile("testdata/textview.go")
	if err != nil {
		t.Error(err)
	}

	prof.Profiling = true
	stt := time.Now()
	lp.Lang.ParseFile(fs)
	prdur := time.Now().Sub(stt)
	fmt.Printf("pi parse: %v\n", prdur)

	prof.Report(time.Millisecond)
	prof.Profiling = false
}

// Note: couldn't get benchmark to do anything reasonable on this one, so just
// using plain test on single iter

func TestGoParse(t *testing.T) {
	stt := time.Now()
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "testdata/textview.go", nil, parser.ParseComments)
	if err != nil {
		t.Error(err)
	}
	prdur := time.Now().Sub(stt)
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
