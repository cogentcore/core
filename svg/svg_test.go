// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/cam/hct"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/gox/dirs"
	"cogentcore.org/core/iox/imagex"
	"cogentcore.org/core/paint"
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
		imagex.Assert(t, sv.Pixels, imfn)
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
		imagex.Assert(t, sv.Pixels, imfn)
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

func TestCoreLogo(t *testing.T) {
	sv := NewSVG(720, 720)
	sv.PhysicalWidth.Px(720)
	sv.PhysicalHeight.Px(720)
	sv.Root.ViewBox.Size.Set(1, 1)

	outer := colors.Scheme.Primary.Base
	hctOuter := hct.FromColor(outer)
	core := hct.New(hctOuter.Hue+180, hctOuter.Chroma, hctOuter.Tone+40)
	// fmt.Println(hct.FromColor(outer))
	// fmt.Println(hct.FromColor(core))

	x := float32(0.53)
	sw := float32(0.27)

	o := NewPath(&sv.Root, "outer")
	o.SetProperty("stroke", colors.AsHex(outer))
	o.SetProperty("stroke-width", sw)
	o.SetProperty("fill", "none")
	o.AddPath(PcM, x, 0.5)
	o.AddPathArc(0.35, 30, 330)
	o.UpdatePathString()

	c := NewCircle(&sv.Root, "core")
	c.Pos.Set(x, 0.5)
	c.Radius = 0.23
	c.SetProperty("fill", colors.AsHex(core))
	c.SetProperty("stroke", "none")

	sv.SaveXML("testdata/logo.svg")

	sv.Background = colors.C(colors.Black)
	sv.Render()
	imagex.Assert(t, sv.Pixels, "logo-black")

	sv.Background = colors.C(colors.White)
	sv.Render()
	imagex.Assert(t, sv.Pixels, "logo-white")
}
