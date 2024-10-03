// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gotosl

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

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/goal/gosl/alignsl"
	"cogentcore.org/core/gpu"
	"golang.org/x/tools/go/packages"
)

// TranslateDir translate all .Go files in given directory to WGSL.
func (st *State) TranslateDir(pf string) error {
	nl := []byte("\n")
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesSizes | packages.NeedTypesInfo}, pf)
	if err != nil {
		return errors.Log(err)
	}
	if len(pkgs) != 1 {
		err := fmt.Errorf("More than one package for path: %v", pf)
		return errors.Log(err)
	}
	pkg := pkgs[0]
	if len(pkg.GoFiles) == 0 {
		err := fmt.Errorf("No Go files found in package: %+v", pkg)
		return errors.Log(err)
	}
	// fmt.Printf("go files: %+v", pkg.GoFiles)
	// return nil, err
	files := pkg.GoFiles

	serr := alignsl.CheckPackage(pkg)

	if serr != nil {
		fmt.Println(serr)
	}

	slrandCopied := false
	sltypeCopied := false

	for _, gofp := range files {
		_, gofn := filepath.Split(gofp)
		wgfn := wgslFile(gofn)
		if st.Config.Debug {
			fmt.Printf("###################################\nTranslating Go file: %s\n", gofn)
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
			fmt.Printf("Warning: File named: %s not found in Loaded package\n", gofn)
			continue
		}

		var buf bytes.Buffer
		pcfg := PrintConfig{GoToSL: st, Mode: printerMode, Tabwidth: tabWidth, ExcludeFunctions: st.ExcludeMap}
		pcfg.Fprint(&buf, pkg, afile)
		// ioutil.WriteFile(filepath.Join(*outDir, fn+".tmp"), buf.Bytes(), 0644)
		slfix, hasSltype, hasSlrand := SlEdits(buf.Bytes())
		if hasSlrand && !slrandCopied {
			hasSltype = true
			if st.Config.Debug {
				fmt.Printf("\tcopying slrand.wgsl to shaders\n")
			}
			st.CopyPackageFile("slrand.wgsl", "cogentcore.org/core/goal/gosl/slrand")
			slrandCopied = true
		}
		if hasSltype && !sltypeCopied {
			if st.Config.Debug {
				fmt.Printf("\tcopying sltype.wgsl to shaders\n")
			}
			st.CopyPackageFile("sltype.wgsl", "cogentcore.org/core/goal/gosl/sltype")
			sltypeCopied = true
		}
		exsl := st.ExtractWGSL(slfix)

		if !st.Config.Keep {
			os.Remove(fpos.Filename)
		}

		slfn := filepath.Join(st.Config.Output, wgfn)
		ioutil.WriteFile(slfn, bytes.Join(exsl, nl), 0644)

		st.SLImportFiles = append(st.SLImportFiles, &File{Name: wgfn, Lines: exsl})
	}
	return nil
}

func (st *State) CompileFile(fn string) error {
	dir, _ := filepath.Abs(st.Config.Output)
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
