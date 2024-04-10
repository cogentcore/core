// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/grows/images"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/xgo/dirs"
	"github.com/stretchr/testify/assert"
)

func TestSVG(t *testing.T) {
	paint.FontLibrary.InitFontPaths(paint.FontPaths...)

	dir := filepath.Join("testdata", "svg")
	files := dirs.ExtFilenames(dir, []string{".svg"})

	for _, fn := range files {
		// if fn != "test2.svg" {
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
	sv.Background = colors.C(colors.White)
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

func TestViewBoxParse(t *testing.T) {
	tests := []string{"none", "xMinYMin", "xMidYMin", "xMaxYMin", "xMinYMax", "xMaxYMax slice"}
	var vb ViewBox
	for _, ts := range tests {
		assert.NoError(t, vb.PreserveAspectRatio.SetString(ts))
		os := vb.PreserveAspectRatio.String()
		if os != ts {
			t.Error("parse fail", os, "!=", ts)
		}
	}
}
