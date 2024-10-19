// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gotosl

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/fsx"
	"golang.org/x/tools/go/packages"
)

// wgslFile returns the file with a ".wgsl" extension
func wgslFile(fn string) string {
	f, _ := fsx.ExtSplit(fn)
	return f + ".wgsl"
}

// bareFile returns the file with no extention
func bareFile(fn string) string {
	f, _ := fsx.ExtSplit(fn)
	return f
}

func ReadFileLines(fn string) ([][]byte, error) {
	nl := []byte("\n")
	buf, err := os.ReadFile(fn)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	lines := bytes.Split(buf, nl)
	return lines, nil
}

func WriteFileLines(fn string, lines [][]byte) error {
	res := bytes.Join(lines, []byte("\n"))
	return os.WriteFile(fn, res, 0644)
}

// HasGoslTag returns true if given file has a //gosl: tag
func (st *State) HasGoslTag(lines [][]byte) bool {
	key := []byte("//gosl:")
	pkg := []byte("package ")
	for _, ln := range lines {
		tln := bytes.TrimSpace(ln)
		if st.Package == "" {
			if bytes.HasPrefix(tln, pkg) {
				st.Package = string(bytes.TrimPrefix(tln, pkg))
			}
		}
		if bytes.HasPrefix(tln, key) {
			return true
		}
	}
	return false
}

func IsGoFile(f fs.DirEntry) bool {
	name := f.Name()
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") && !f.IsDir()
}

func IsWGSLFile(f fs.DirEntry) bool {
	name := f.Name()
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".wgsl") && !f.IsDir()
}

// ProjectFiles gets the files in the current directory.
func (st *State) ProjectFiles() {
	fls := fsx.Filenames(".", ".go")
	st.GoFiles = make(map[string]*File)
	st.GoVarsFiles = make(map[string]*File)
	for _, fn := range fls {
		fl := &File{Name: fn}
		var err error
		fl.Lines, err = ReadFileLines(fn)
		if err != nil {
			continue
		}
		if !st.HasGoslTag(fl.Lines) {
			continue
		}
		st.GoFiles[fn] = fl
		st.ImportFiles(fl.Lines)
	}
}

// ImportFiles checks the given content for //gosl:import tags
// and imports the package if so.
func (st *State) ImportFiles(lines [][]byte) {
	key := []byte("//gosl:import ")
	for _, ln := range lines {
		tln := bytes.TrimSpace(ln)
		if !bytes.HasPrefix(tln, key) {
			continue
		}
		impath := strings.TrimSpace(string(tln[len(key):]))
		if impath[0] == '"' {
			impath = impath[1:]
		}
		if impath[len(impath)-1] == '"' {
			impath = impath[:len(impath)-1]
		}
		_, ok := st.GoImports[impath]
		if ok {
			continue
		}
		var pkgs []*packages.Package
		var err error
		pkgs, err = packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles}, impath)
		if err != nil {
			fmt.Println(err)
			continue
		}
		pfls := make(map[string]*File)
		st.GoImports[impath] = pfls
		pkg := pkgs[0]
		gofls := pkg.GoFiles
		if len(gofls) == 0 {
			fmt.Printf("WARNING: no go files found in path: %s\n", impath)
		}
		for _, gf := range gofls {
			lns, err := ReadFileLines(gf)
			if err != nil {
				continue
			}
			if !st.HasGoslTag(lns) {
				continue
			}
			_, fo := filepath.Split(gf)
			pfls[fo] = &File{Name: fo, Lines: lns}
			st.ImportFiles(lns)
			// fmt.Printf("added file: %s from package: %s\n", gf, impath)
		}
		st.GoImports[impath] = pfls
	}
}

// RemoveGenFiles removes .go, .wgsl, .spv files in shader generated dir
func RemoveGenFiles(dir string) {
	err := filepath.WalkDir(dir, func(path string, f fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if IsGoFile(f) || IsWGSLFile(f) {
			os.Remove(path)
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

// CopyPackageFile copies given file name from given package path
// into the current imports directory.
// e.g., "slrand.wgsl", "cogentcore.org/core/goal/gosl/slrand"
func (st *State) CopyPackageFile(fnm, packagePath string) error {
	tofn := filepath.Join(st.ImportsDir, fnm)
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles}, packagePath)
	if err != nil {
		fmt.Println(err)
		return err
	}
	if len(pkgs) != 1 {
		err = fmt.Errorf("%s package not found", packagePath)
		fmt.Println(err)
		return err
	}
	pkg := pkgs[0]
	var fn string
	if len(pkg.GoFiles) > 0 {
		fn = pkg.GoFiles[0]
	} else if len(pkg.OtherFiles) > 0 {
		fn = pkg.GoFiles[0]
	} else {
		err = fmt.Errorf("No files found in package: %s", packagePath)
		fmt.Println(err)
		return err
	}
	dir, _ := filepath.Split(fn)
	fmfn := filepath.Join(dir, fnm)
	lines, err := CopyFile(fmfn, tofn)
	if err == nil {
		lines = SlRemoveComments(lines)
		st.SLImportFiles = append(st.SLImportFiles, &File{Name: fnm, Lines: lines})
	}
	return nil
}

func CopyFile(src, dst string) ([][]byte, error) {
	lines, err := ReadFileLines(src)
	if err != nil {
		return lines, err
	}
	err = WriteFileLines(dst, lines)
	if err != nil {
		return lines, err
	}
	return lines, err
}
