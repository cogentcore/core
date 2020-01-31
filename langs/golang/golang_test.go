// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"
	"go/parser"
	"go/token"
	"testing"
	"time"

	"github.com/goki/pi/filecat"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
	"github.com/goki/prof"
)

func init() {
	pi.LangSupport.OpenStd()
}

func TestParse(t *testing.T) {
	// t.Skip("todo: reenable soon")
	lp, _ := pi.LangSupport.Props(filecat.Go)
	pr := lp.Lang.Parser()
	pr.ReportErrs = true

	fs := pi.NewFileStates("testdata/textview.go", "", filecat.Go)
	txt, err := lex.OpenFileBytes(fs.Filename) // and other stuff
	if err != nil {
		t.Error(err)
	}

	prof.Profiling = true
	stt := time.Now()
	lp.Lang.ParseFile(fs, txt)
	prdur := time.Now().Sub(stt)
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
