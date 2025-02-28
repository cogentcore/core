// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

func TestSVG(t *testing.T) {
	dir := filepath.Join("testdata", "svg")
	files := fsx.Filenames(dir, ".svg")

	for _, fn := range files {
		// if fn != "text-test.svg" {
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
		imagex.Assert(t, sv.RenderImage(), imfn)
	}
}

func TestViewBox(t *testing.T) {
	dir := filepath.Join("testdata", "svg")
	sfn := "fig_necker_cube.svg"
	file := filepath.Join(dir, sfn)
	tests := []string{"none", "xMinYMin", "xMidYMid", "xMaxYMax", "xMaxYMax slice"}
	sv := NewSVG(640, 480)
	sv.Background = colors.Uniform(colors.White)
	err := sv.OpenXML(file)
	if err != nil {
		t.Error("error opening xml:", err)
		return
	}
	for _, ts := range tests {
		// if ts != "xMinYMin" {
		// 	continue
		// }
		fpre := strings.TrimSuffix(sfn, ".svg")
		sv.Root.ViewBox.PreserveAspectRatio.SetString(ts)
		sv.Render()

		fnm := fmt.Sprintf("%s_%s", fpre, ts)
		imfn := filepath.Join("png", filepath.Join("viewbox", fnm))
		imagex.Assert(t, sv.RenderImage(), imfn)
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
	sv.PhysicalWidth.Dp(256)
	sv.PhysicalHeight.Dp(256)
	sv.Root.ViewBox.Size.Set(1, 1)

	outer := colors.Scheme.Primary.Base // #005BC0
	hctOuter := hct.FromColor(colors.ToUniform(outer))
	core := hct.New(hctOuter.Hue+180, hctOuter.Chroma, hctOuter.Tone+40) // #FBBD0E

	x := float32(0.53)
	sw := float32(0.40)

	o := NewPath(sv.Root)
	o.SetProperty("stroke", colors.AsHex(colors.ToUniform(outer)))
	o.SetProperty("stroke-width", sw)
	o.SetProperty("fill", "none")
	o.Data.CircularArc(x, 0.5, 0.35, math32.DegToRad(30), math32.DegToRad(330))
	o.UpdatePathString()

	c := NewCircle(sv.Root)
	c.Pos.Set(x, 0.5)
	c.Radius = 0.23
	c.SetProperty("fill", colors.AsHex(core))
	c.SetProperty("stroke", "none")

	sv.SaveXML("testdata/logo.svg")

	sv.Background = colors.Uniform(colors.Black)
	sv.Render()
	imagex.Assert(t, sv.RenderImage(), "logo-black")

	sv.Background = colors.Uniform(colors.White)
	sv.Render()
	imagex.Assert(t, sv.RenderImage(), "logo-white")
}
