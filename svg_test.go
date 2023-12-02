// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"goki.dev/girl/paint"
	"goki.dev/glop/dirs"
	"goki.dev/grows/images"
)

func TestSVG(t *testing.T) {
	paint.FontLibrary.InitFontPaths(paint.FontPaths...)

	dir := filepath.Join("testdata", "svg")
	files := dirs.ExtFileNames(dir, []string{".svg"})

	for _, fn := range files {
		// if fn != "fig_bp_compute_delta.svg" {
		// 	continue
		// }
		sv := NewSVG(640, 480)
		svfn := filepath.Join(dir, fn)
		err := sv.OpenXML(svfn)
		if err != nil {
			fmt.Println("error opening xml:", err)
			continue
		}
		// fmt.Println(sv.Root.ViewBox)
		sv.SetNormXForm()
		sv.Render()
		imfn := filepath.Join("png", strings.TrimSuffix(fn, ".svg"))
		images.Assert(t, sv.Pixels, imfn)
	}
}
