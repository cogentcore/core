// Copyright (c) 2018, The gide / GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/ki/dirs"
	"github.com/goki/pi/filecat"
	_ "github.com/goki/pi/langs"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
)

var Excludes []string

func main() {
	var path string
	var recurse bool
	var excl string

	pi.LangSupport.OpenStd()

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\ne.g., on mac, to run all of the std go files except cmd which is large and slow:\npi -r -ex cmd /usr/local/Cellar/go/1.11.3/libexec/src\n\n")
	}

	// process command args
	flag.StringVar(&path, "path", "", "path to open -- can be to a directory or a filename within the directory -- or just last arg without a flag")
	flag.BoolVar(&recurse, "r", false, "recursive -- apply to subdirectories")
	flag.StringVar(&excl, "ex", "", "comma-separated list of directory names to exclude, for recursive case")
	flag.Parse()
	if path == "" {
		if flag.NArg() > 0 {
			path = flag.Arg(0)
		} else {
			path = "."
		}
	}
	Excludes = strings.Split(excl, ",")

	// todo: assuming go for now
	if recurse {
		DoGoRecursive(path)
	} else {
		DoGoPath(path)
	}
}

func DoGoPath(path string) {
	fmt.Printf("Processing path: %v\n", path)
	lp, _ := pi.LangSupport.Props(filecat.Go)
	pr := lp.Lang.Parser()
	pr.ReportErrs = true
	fs := pi.NewFileState()
	pkgsym := lp.Lang.ParseDir(fs, path, pi.LangDirOpts{Rebuild: true})
	if pkgsym != nil {
		syms.SaveSymDoc(pkgsym, filecat.Go, path)
	}
}

func DoGoRecursive(path string) {
	DoGoPath(path)
	drs := dirs.Dirs(path)
outer:
	for _, dr := range drs {
		if dr == "testdata" {
			continue
		}
		for _, ex := range Excludes {
			if dr == ex {
				continue outer
			}
		}
		sp := filepath.Join(path, dr)
		DoGoRecursive(sp)
	}
}
