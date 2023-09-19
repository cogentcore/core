// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"goki.dev/girl/girl"
	"goki.dev/girl/gist"
	"goki.dev/glop/dirs"
	"goki.dev/svg"
)

func main() {
	prefs := &gist.Prefs{}
	prefs.Defaults()
	gist.ThePrefs = prefs // text rendering depends on this

	// in GoGi, oswin.TheApp gives you default font paths per-platform.

	// mac:
	girl.FontLibrary.InitFontPaths("/System/Library/Fonts", "/Library/Fonts")

	dir := "./svgs"
	out := "./testdata"
	files := dirs.ExtFileNames(dir, []string{".svg"})

	for _, fn := range files {
		if fn != "fig_bp_compute_delta.svg" {
			continue
		}
		// if fn != "fig_cortex_lobes.svg" {
		// 	continue
		// }
		fmt.Println(fn)
		sv := svg.NewSVG(640, 480)
		svfn := filepath.Join(dir, fn)
		sv.OpenXML(svfn)
		sv.SetNormXForm()
		sv.Render()
		imfn := filepath.Join(out, strings.TrimSuffix(fn, ".svg")+".png")
		svg.SaveImage(imfn, sv.Pixels)
	}
}
