// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gotosl

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/goal/gosl/alignsl"
	"golang.org/x/exp/maps"
	"golang.org/x/tools/go/packages"
)

// TranslateDir translate all .Go files in given directory to WGSL.
func (st *State) TranslateDir(pf string) error {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesSizes | packages.NeedTypesInfo}, pf)
	// pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, pf)
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

	st.FuncGraph = make(map[string]*Function)
	st.GetFuncGraph = true

	doFile := func(gofp string, buf *bytes.Buffer) {
		_, gofn := filepath.Split(gofp)
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
			return
		}

		pcfg := PrintConfig{GoToSL: st, Mode: printerMode, Tabwidth: tabWidth, ExcludeFunctions: st.ExcludeMap}
		pcfg.Fprint(buf, pkg, afile)
		if !st.GetFuncGraph && !st.Config.Keep {
			os.Remove(fpos.Filename)
		}
	}

	// first pass is just to get the call graph:
	for fn := range st.GoVarsFiles { // do varsFiles first!!
		var buf bytes.Buffer
		doFile(fn, &buf)
	}
	for _, gofp := range files {
		_, gofn := filepath.Split(gofp)
		if _, ok := st.GoVarsFiles[gofn]; ok {
			continue
		}
		var buf bytes.Buffer
		doFile(gofp, &buf)
	}

	// st.PrintFuncGraph()

	doKernelFile := func(fname string, lines [][]byte) ([][]byte, bool, bool) {
		_, gofn := filepath.Split(fname)
		var buf bytes.Buffer
		doFile(fname, &buf)
		slfix, hasSltype, hasSlrand := SlEdits(buf.Bytes())
		slfix = SlRemoveComments(slfix)
		exsl := st.ExtractWGSL(slfix)
		lines = append(lines, []byte(""))
		lines = append(lines, []byte(fmt.Sprintf("//////// import: %q", gofn)))
		lines = append(lines, exsl...)
		return lines, hasSltype, hasSlrand
	}

	// next pass is per kernel
	st.GetFuncGraph = false
	sys := maps.Keys(st.Systems)
	sort.Strings(sys)
	for _, snm := range sys {
		sy := st.Systems[snm]
		kns := maps.Keys(sy.Kernels)
		sort.Strings(kns)
		for _, knm := range kns {
			kn := sy.Kernels[knm]
			st.KernelFuncs = st.AllFuncs(kn.Name)
			if st.KernelFuncs == nil {
				continue
			}
			st.CopyPackageFile("slrand.wgsl", "cogentcore.org/core/goal/gosl/slrand")
			st.CopyPackageFile("sltype.wgsl", "cogentcore.org/core/goal/gosl/sltype")
			var hasSltype, hasSlrand, hasT, hasR bool
			avars := st.AtomicVars(st.KernelFuncs)
			// if st.Config.Debug {
			fmt.Printf("###################################\nTranslating Kernel file: %s\n", kn.Name)
			// }
			hdr := st.GenKernelHeader(sy, kn, avars)
			lines := bytes.Split([]byte(hdr), []byte("\n"))
			for fn := range st.GoVarsFiles { // do varsFiles first!!
				lines, hasT, hasR = doKernelFile(fn, lines)
				if hasT {
					hasSltype = true
				}
				if hasR {
					hasSlrand = true
				}
			}
			for _, gofp := range files {
				_, gofn := filepath.Split(gofp)
				if _, ok := st.GoVarsFiles[gofn]; ok {
					continue
				}
				lines, hasT, hasR = doKernelFile(gofp, lines)
				if hasT {
					hasSltype = true
				}
				if hasR {
					hasSlrand = true
				}
			}
			// this is not working
			// if hasSlrand {
			// 	st.CopyPackageFile("slrand.wgsl", "cogentcore.org/core/goal/gosl/slrand")
			// 	hasSltype = true
			// }
			// if hasSltype {
			// 	st.CopyPackageFile("sltype.wgsl", "cogentcore.org/core/goal/gosl/sltype")
			// }
			_ = hasSltype
			_ = hasSlrand
			for _, im := range st.SLImportFiles {
				lines = append(lines, []byte(""))
				lines = append(lines, []byte(fmt.Sprintf("//////// import: %q", im.Name)))
				lines = append(lines, im.Lines...)
			}
			kn.Lines = lines
			kfn := kn.Name + ".wgsl"
			fn := filepath.Join(st.Config.Output, kfn)
			kn.Filename = fn
			WriteFileLines(fn, lines)
			st.CompileFile(kfn)
		}
	}

	return nil
}

var (
	nagaWarned = false
	tintWarned = false
)

func (st *State) CompileFile(fn string) error {
	dir, _ := filepath.Abs(st.Config.Output)
	if _, err := exec.LookPath("naga"); err == nil {
		// cmd := exec.Command("naga", "--compact", fn, fn) // produces some pretty weird code actually
		cmd := exec.Command("naga", fn)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		fmt.Printf("\n-----------------------------------------------------\nnaga output for: %s\n%s", fn, out)
		if err != nil {
			log.Println(err)
			return err
		}
	} else {
		if !nagaWarned {
			fmt.Println("\nImportant: you should install the 'naga' WGSL compiler from https://github.com/gfx-rs/wgpu to get immediate validation")
			nagaWarned = true
		}
	}
	if _, err := exec.LookPath("tint"); err == nil {
		cmd := exec.Command("tint", "--validate", "--format", "wgsl", "-o", "/dev/null", fn)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		fmt.Printf("\n-----------------------------------------------------\ntint output for: %s\n%s", fn, out)
		if err != nil {
			log.Println(err)
			return err
		}
	} else {
		if !tintWarned {
			fmt.Println("\nImportant: you should install the 'tint' WGSL compiler from https://dawn.googlesource.com/dawn/ to get immediate validation")
			tintWarned = true
		}
	}

	return nil
}
