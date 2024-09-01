// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// heavily modified from go src/cmd/gofmt/internal.go:

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cogentcore.org/core/gpu/gosl/alignsl"
	"cogentcore.org/core/gpu/gosl/slprint"
	"golang.org/x/tools/go/packages"
)

// does all the file processing
func ProcessFiles(paths []string) (map[string][]byte, error) {
	fls := FilesFromPaths(paths)
	gosls := ExtractGoFiles(fls) // extract Go files to shader/*.go

	wgslFiles := []string{}
	for _, fn := range fls {
		if strings.HasSuffix(fn, ".wgsl") {
			wgslFiles = append(wgslFiles, fn)
		}
	}

	pf := "./" + *outDir
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes}, pf)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if len(pkgs) != 1 {
		err := fmt.Errorf("More than one package for path: %v", pf)
		log.Println(err)
		return nil, err
	}
	pkg := pkgs[0]

	if len(pkg.GoFiles) == 0 {
		err := fmt.Errorf("No Go files found in package: %+v", pkg)
		log.Println(err)
		return nil, err
	}
	// fmt.Printf("go files: %+v", pkg.GoFiles)
	// return nil, err

	// map of files with a main function that needs to be compiled
	needsCompile := map[string]bool{}

	serr := alignsl.CheckPackage(pkg)
	if serr != nil {
		fmt.Println(serr)
	}

	slrandCopied := false
	for fn := range gosls {
		gofn := fn + ".go"
		if *debug {
			fmt.Printf("###################################\nProcessing Go file: %s\n", gofn)
		}

		var afile *ast.File
		var fpos token.Position
		for _, sy := range pkg.Syntax {
			pos := pkg.Fset.Position(sy.Package)
			_, posfn := filepath.Split(pos.Filename)
			if posfn == gofn {
				fpos = pos
				afile = sy
				break
			}
		}
		if afile == nil {
			fmt.Printf("Warning: File named: %s not found in processed package\n", gofn)
			continue
		}

		var buf bytes.Buffer
		cfg := slprint.Config{Mode: printerMode, Tabwidth: tabWidth, ExcludeFunctions: excludeFunctionMap}
		// cfg.Fprint(&buf, pkg, fpos, afile)
		cfg.Fprint(&buf, pkg.Fset, afile)
		// ioutil.WriteFile(filepath.Join(*outDir, fn+".tmp"), buf.Bytes(), 0644)
		slfix, hasSlrand := SlEdits(buf.Bytes())
		if hasSlrand && !slrandCopied {
			if *debug {
				fmt.Printf("\tcopying slrand.wgsl to shaders\n")
			}
			CopySlrand()
			slrandCopied = true
		}
		exsl, hasMain := ExtractWGSL(slfix)
		gosls[fn] = exsl

		if hasMain {
			needsCompile[fn] = true
		}
		if !*keepTmp {
			os.Remove(fpos.Filename)
		}

		// add wgsl code
		for _, hlfn := range wgslFiles {
			if fn+".wgsl" != hlfn {
				continue
			}
			buf, err := os.ReadFile(hlfn)
			if err != nil {
				fmt.Println(err)
				continue
			}
			exsl = append(exsl, []byte(fmt.Sprintf("\n// from file: %s\n", hlfn))...)
			exsl = append(exsl, buf...)
			gosls[fn] = exsl
			needsCompile[fn] = true // assume any standalone has main
			break
		}

		// upfn := strings.ToUpper(fn)
		// once := fmt.Sprintf("#ifndef __%s_WGSL__\n#define __%s_WGSL__\n\n", upfn, upfn)
		// exsl = append([]byte(once), exsl...)
		// oncend := fmt.Sprintf("#endif // __%s_WGSL__\n", upfn)
		// exsl = append(exsl, []byte(oncend)...)

		slfn := filepath.Join(*outDir, fn+".wgsl")
		ioutil.WriteFile(slfn, exsl, 0644)
	}

	// check for wgsl files that had no go equivalent
	for _, hlfn := range wgslFiles {
		hasGo := false
		for fn := range gosls {
			if fn+".wgsl" == hlfn {
				hasGo = true
				break
			}
		}
		if hasGo {
			continue
		}
		_, hlfno := filepath.Split(hlfn) // could be in a subdir
		tofn := filepath.Join(*outDir, hlfno)
		CopyFile(hlfn, tofn)
		fn := strings.TrimSuffix(hlfno, ".wgsl")
		needsCompile[fn] = true // assume any standalone wgsl is a main
	}

	for fn := range needsCompile {
		CompileFile(fn + ".wgsl")
	}
	return gosls, nil
}

func CompileFile(fn string) error {
	cmd := exec.Command("naga", fn)
	cmd.Dir, _ = filepath.Abs(*outDir)
	out, err := cmd.CombinedOutput()
	fmt.Printf("\n-----------------------------------------------------\nnaga output for: %s\n%s", fn, out)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
