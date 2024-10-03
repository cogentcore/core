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

func (st *State) WriteFileLines(fn string, lines [][]byte) error {
	outfn := filepath.Join(st.Config.Output, fn)
	res := bytes.Join(lines, []byte("\n"))
	return os.WriteFile(outfn, res, 0644)
}

// HasGoslTag returns true if given file has a //gosl: tag
func HasGoslTag(lines [][]byte) bool {
	key := []byte("//gosl:")
	for _, ln := range lines {
		tln := bytes.TrimSpace(ln)
		if bytes.HasPrefix(tln, key) {
			return true
		}
	}
	return false
}

// LoadedPackageNames are single prefix names of packages that were
// loaded in the list of files to process
var LoadedPackageNames = map[string]bool{}

func IsGoFile(f fs.DirEntry) bool {
	name := f.Name()
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") && !f.IsDir()
}

func IsWGSLFile(f fs.DirEntry) bool {
	name := f.Name()
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".wgsl") && !f.IsDir()
}

func IsSPVFile(f fs.DirEntry) bool {
	name := f.Name()
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".spv") && !f.IsDir()
}

// ProjectFiles gets the files in the current directory.
func (st *State) ProjectFiles() {
	fls := fsx.Filenames(".", ".go")
	st.Files = make(map[string]*File)
	for _, fn := range fls {
		fl := &File{Name: fn}
		var err error
		fl.Lines, err = ReadFileLines(fn)
		if err != nil {
			continue
		}
		if !HasGoslTag(fl.Lines) {
			continue
		}
		st.Files[fn] = fl
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
		_, ok := st.Imports[impath]
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
		st.Imports[impath] = pfls
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
			if !HasGoslTag(lns) {
				continue
			}
			_, fo := filepath.Split(gf)
			pfls[fo] = &File{Name: fo, Lines: lns}
			st.ImportFiles(lns)
			// fmt.Printf("added file: %s from package: %s\n", gf, impath)
		}
		st.Imports[impath] = pfls
	}
}

// RemoveGenFiles removes .go, .wgsl, .spv files in shader generated dir
func RemoveGenFiles(dir string) {
	err := filepath.WalkDir(dir, func(path string, f fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if IsGoFile(f) || IsWGSLFile(f) || IsSPVFile(f) {
			os.Remove(path)
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}
