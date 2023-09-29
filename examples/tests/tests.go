// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goki.dev/girl/girl"
	"goki.dev/glop/dirs"
	"goki.dev/svg"
)

func main() {
	girl.FontLibrary.InitFontPaths(girl.FontPaths...)

	dir := "./svgs"
	out := "./testdata"
	err := os.MkdirAll(out, 0755)
	if err != nil {
		panic("error creating testdata directory if it doesn't already exist")
	}
	files := dirs.ExtFileNames(dir, []string{".svg"})

	for _, fn := range files {
		// if fn != "fig_bp_compute_delta.svg" {
		// 	continue
		// }
		fmt.Println(fn)
		sv := svg.NewSVG(640, 480)
		svfn := filepath.Join(dir, fn)
		err := sv.OpenXML(svfn)
		if err != nil {
			fmt.Println("error opening xml:", err)
			continue
		}
		// fmt.Println(sv.Root.ViewBox)
		sv.SetNormXForm()
		sv.Render()
		imfn := filepath.Join(out, strings.TrimSuffix(fn, ".svg")+".png")
		svg.SaveImage(imfn, sv.Pixels)
	}
}
