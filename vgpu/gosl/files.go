// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// LoadedPackageNames are single prefix names of packages that were
// loaded in the list of files to process
var LoadedPackageNames = map[string]bool{}

func IsGoFile(f fs.DirEntry) bool {
	name := f.Name()
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") && !f.IsDir()
}

func IsHLSLFile(f fs.DirEntry) bool {
	name := f.Name()
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".hlsl") && !f.IsDir()
}

func IsSPVFile(f fs.DirEntry) bool {
	name := f.Name()
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".spv") && !f.IsDir()
}

func AddFile(fn string, fls []string, procd map[string]bool) []string {
	if _, has := procd[fn]; has {
		return fls
	}
	fls = append(fls, fn)
	procd[fn] = true
	dir, _ := filepath.Split(fn)
	if dir != "" {
		dir = dir[:len(dir)-1]
		pd, sd := filepath.Split(dir)
		if pd != "" {
			dir = sd
		}
		if !(dir == "math32") {
			if _, has := LoadedPackageNames[dir]; !has {
				LoadedPackageNames[dir] = true
				// fmt.Printf("package: %s\n", dir)
			}
		}
	}
	return fls
}

// FilesFromPaths processes all paths and returns a full unique list of files
// for subsequent processing.
func FilesFromPaths(paths []string) []string {
	fls := make([]string, 0, len(paths))
	procd := make(map[string]bool)
	for _, path := range paths {
		switch info, err := os.Stat(path); {
		case err != nil:
			var pkgs []*packages.Package
			dir, fl := filepath.Split(path)
			if dir != "" && fl != "" && strings.HasSuffix(fl, ".go") {
				pkgs, err = packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles}, dir)
			} else {
				fl = ""
				pkgs, err = packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles}, path)
			}
			if err != nil {
				fmt.Println(err)
				continue
			}
			pkg := pkgs[0]
			gofls := pkg.GoFiles
			if len(gofls) == 0 {
				fmt.Printf("WARNING: no go files found in path: %s\n", path)
			}
			if fl != "" {
				for _, gf := range gofls {
					if strings.HasSuffix(gf, fl) {
						fls = AddFile(gf, fls, procd)
						// fmt.Printf("added file: %s from package: %s\n", gf, path)
						break
					}
				}
			} else {
				for _, gf := range gofls {
					fls = AddFile(gf, fls, procd)
					// fmt.Printf("added file: %s from package: %s\n", gf, path)
				}
			}
		case !info.IsDir():
			path := path
			fls = AddFile(path, fls, procd)
		default:
			// Directories are walked, ignoring non-Go, non-HLSL files.
			err := filepath.WalkDir(path, func(path string, f fs.DirEntry, err error) error {
				if err != nil || !(IsGoFile(f) || IsHLSLFile(f)) {
					return err
				}
				_, err = f.Info()
				if err != nil {
					return nil
				}
				fls = AddFile(path, fls, procd)
				return nil
			})
			if err != nil {
				log.Println(err)
			}
		}
	}
	return fls
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func CopySlrand() error {
	hdr := "slrand.hlsl"
	tofn := filepath.Join(*outDir, hdr)

	pnm := "github.com/emer/gosl/v2/slrand"

	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles}, pnm)
	if err != nil {
		fmt.Println(err)
		return err
	}
	if len(pkgs) != 1 {
		err = fmt.Errorf("%s package not found", pnm)
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
		err = fmt.Errorf("No files found in package: %s", pnm)
		fmt.Println(err)
		return err
	}
	dir, _ := filepath.Split(fn)
	// dir = filepath.Join(dir, "slrand")
	fmfn := filepath.Join(dir, hdr)
	CopyFile(fmfn, tofn)
	return nil
}

// RemoveGenFiles removes .go, .hlsl, .spv files in shader generated dir
func RemoveGenFiles(dir string) {
	err := filepath.WalkDir(dir, func(path string, f fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if IsGoFile(f) || IsHLSLFile(f) || IsSPVFile(f) {
			os.Remove(path)
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}
