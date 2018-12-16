// Copyright (c) 2018, The gide / GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"os"

	"github.com/goki/gi/filecat"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
)

func main() {
	var path string

	pi.LangSupport.OpenStd()

	// process command args
	if len(os.Args) > 1 {
		flag.StringVar(&path, "path", "", "path to open -- can be to a directory or a filename within the directory")
		// todo: other args?
		flag.Parse()
		if path == "" {
			if flag.NArg() > 0 {
				path = flag.Arg(0)
			} else {
				path = "."
			}
		}
	}

	// todo: assuming go for now
	lp, _ := pi.LangSupport.Props(filecat.Go)
	pkgsym := lp.Lang.ParseDir(path, pi.LangDirOpts{Rebuild: true})
	if pkgsym != nil {
		syms.SaveSymDoc(pkgsym, path)
	}
}
