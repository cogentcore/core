// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package golang

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

// packageGoFile returns the first Go file of the package with the given
// import path, using [packages.Load], which deals with GOPATH vs. modules, etc.
func packageGoFile(path string) (string, bool) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles}, path)
	if err != nil {
		// this is too many errors!
		// log.Println(err)
		return "", false
	}
	if len(pkgs) != 1 {
		fmt.Printf("More than one package for path: %v\n", path)
		return "", false
	}
	pkg := pkgs[0]
	if len(pkg.GoFiles) == 0 {
		// fmt.Printf("No Go files found in package: %v\n", path)
		return "", false
	}
	return pkg.GoFiles[0], true
}
