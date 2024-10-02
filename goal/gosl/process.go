// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/gpu"
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
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesSizes | packages.NeedTypesInfo}, pf)
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
	sltypeCopied := false
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
		cfg.Fprint(&buf, pkg, afile)
		// ioutil.WriteFile(filepath.Join(*outDir, fn+".tmp"), buf.Bytes(), 0644)
		slfix, hasSltype, hasSlrand := SlEdits(buf.Bytes())
		if hasSlrand && !slrandCopied {
			hasSltype = true
			if *debug {
				fmt.Printf("\tcopying slrand.wgsl to shaders\n")
			}
			CopyPackageFile("slrand.wgsl", "cogentcore.org/core/gpu/gosl/slrand")
			slrandCopied = true
		}
		if hasSltype && !sltypeCopied {
			if *debug {
				fmt.Printf("\tcopying sltype.wgsl to shaders\n")
			}
			CopyPackageFile("sltype.wgsl", "cogentcore.org/core/gpu/gosl/sltype")
			sltypeCopied = true
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
		for _, slfn := range wgslFiles {
			if fn+".wgsl" != slfn {
				continue
			}
			buf, err := os.ReadFile(slfn)
			if err != nil {
				fmt.Println(err)
				continue
			}
			exsl = append(exsl, []byte(fmt.Sprintf("\n// from file: %s\n", slfn))...)
			exsl = append(exsl, buf...)
			gosls[fn] = exsl
			needsCompile[fn] = true // assume any standalone has main
			break
		}

		slfn := filepath.Join(*outDir, fn+".wgsl")
		ioutil.WriteFile(slfn, exsl, 0644)
	}

	// check for wgsl files that had no go equivalent
	for _, slfn := range wgslFiles {
		hasGo := false
		for fn := range gosls {
			if fn+".wgsl" == slfn {
				hasGo = true
				break
			}
		}
		if hasGo {
			continue
		}
		_, slfno := filepath.Split(slfn) // could be in a subdir
		tofn := filepath.Join(*outDir, slfno)
		CopyFile(slfn, tofn)
		fn := strings.TrimSuffix(slfno, ".wgsl")
		needsCompile[fn] = true // assume any standalone wgsl is a main
	}

	for fn := range needsCompile {
		CompileFile(fn + ".wgsl")
	}
	return gosls, nil
}

func CompileFile(fn string) error {
	dir, _ := filepath.Abs(*outDir)
	fsys := os.DirFS(dir)
	b, err := fs.ReadFile(fsys, fn)
	if errors.Log(err) != nil {
		return err
	}
	is := gpu.IncludeFS(fsys, "", string(b))
	ofn := filepath.Join(dir, fn)
	err = os.WriteFile(ofn, []byte(is), 0666)
	if errors.Log(err) != nil {
		return err
	}
	cmd := exec.Command("naga", fn)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	fmt.Printf("\n-----------------------------------------------------\nnaga output for: %s\n%s", fn, out)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
