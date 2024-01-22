// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/glop/dirs"
	"cogentcore.org/core/grows/images"
	"cogentcore.org/core/paint"
)

func TestSVG(t *testing.T) {
	paint.FontLibrary.InitFontPaths(paint.FontPaths...)

	dir := filepath.Join("testdata", "svg")
	files := dirs.ExtFilenames(dir, []string{".svg"})

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
		sv.Render()
		imfn := filepath.Join("png", strings.TrimSuffix(fn, ".svg"))
		images.Assert(t, sv.Pixels, imfn)
	}
}

func TestViewBox(t *testing.T) {
	paint.FontLibrary.InitFontPaths(paint.FontPaths...)

	dir := filepath.Join("testdata", "svg")
	sfn := "fig_necker_cube.svg"
	file := filepath.Join(dir, sfn)

	tests := []string{"none", "xMinYMin", "xMidYMid", "xMaxYMax", "xMaxYMax slice"}
	sv := NewSVG(640, 480)
	sv.Fill = true
	err := sv.OpenXML(file)
	if err != nil {
		t.Error("error opening xml:", err)
		return
	}
	fpre := strings.TrimSuffix(sfn, ".svg")
	for _, ts := range tests {
		sv.Root.ViewBox.PreserveAspectRatio.SetString(ts)
		sv.Render()
		fnm := fmt.Sprintf("%s_%s", fpre, ts)
		imfn := filepath.Join("png", fnm)
		images.Assert(t, sv.Pixels, imfn)
	}
}
