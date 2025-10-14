// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg_test

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/math32"
	_ "cogentcore.org/core/paint/renderers" // installs default renderer
	. "cogentcore.org/core/svg"
	"github.com/go-text/typesetting/font"
	"github.com/stretchr/testify/assert"
)

func RunTest(t *testing.T, width, height int, dir, fname string) {
	sv := NewSVG(math32.Vec2(float32(width), float32(height)))
	svfn := filepath.Join("testdata", dir, fname)
	err := sv.OpenXML(svfn)
	assert.NoError(t, err)
	img := sv.RenderImage()
	imfn := filepath.Join(dir, "png", strings.TrimSuffix(fname, ".svg"))
	// fmt.Println(svfn, imfn)
	imagex.Assert(t, img, imfn)
}

func TestSVG(t *testing.T) {
	dir := "svg"
	files := fsx.Filenames(filepath.Join("testdata", dir), ".svg")

	for _, fn := range files {
		// if fn != "fig_neuron_as_detect.svg" {
		// 	continue
		// }
		RunTest(t, 640, 480, dir, fn)
	}
}

func TestViewBox(t *testing.T) {
	dir := filepath.Join("testdata", "svg")
	sfn := "fig_necker_cube.svg"
	file := filepath.Join(dir, sfn)
	tests := []string{"none", "xMinYMin", "xMidYMid", "xMaxYMax", "xMaxYMax slice"}
	sv := NewSVG(math32.Vec2(640, 480))
	sv.Background = colors.Uniform(colors.White)
	err := sv.OpenXML(file)
	assert.NoError(t, err)
	for _, ts := range tests {
		// if ts != "xMinYMin" {
		// 	continue
		// }
		fpre := strings.TrimSuffix(sfn, ".svg")
		sv.Root.ViewBox.PreserveAspectRatio.SetString(ts)
		img := sv.RenderImage()

		fnm := fmt.Sprintf("%s_%s", fpre, ts)
		imfn := filepath.Join("svg", "png", "viewbox", fnm)
		// fmt.Println(imfn)
		imagex.Assert(t, img, imfn)
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
	sv := NewSVG(math32.Vec2(720, 720))
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
	img := sv.RenderImage()
	imagex.Assert(t, img, "logo-black")

	sv.Background = colors.Uniform(colors.White)
	img = sv.RenderImage()
	imagex.Assert(t, img, "logo-white")
}

func TestEmoji(t *testing.T) {
	// dir := "noto-emoji"
	dir := "emoji-bad"
	// dir := "font-emoji-src"
	files := fsx.Filenames(filepath.Join("testdata", dir), ".svg")

	for _, fn := range files {
		// if fn != "femoji-23.svg" {
		// 	continue
		// }
		RunTest(t, 512, 512, dir, fn)
	}
}

func TestFontEmoji(t *testing.T) {
	// t.Skip("special-case testing -- requires noto-emoji file")
	// dir := filepath.Join("testdata", "noto-emoji")
	os.MkdirAll("testdata/font-emoji-src", 0777)
	fname := "/Library/Fonts/NotoColorEmoji-Regular.ttf"
	b, err := os.ReadFile(fname)
	assert.NoError(t, err)
	faces, err := font.ParseTTC(bytes.NewReader(b))
	assert.NoError(t, err)
	face := faces[0]
	ctr := 0
	for r := rune(0); r < math.MaxInt32; r++ {
		gid, has := face.NominalGlyph(r)
		if !has {
			continue
		}
		data := face.GlyphData(gid)
		gd, ok := data.(font.GlyphSVG)
		if !ok {
			continue
		}
		fn := fmt.Sprintf("femoji-%x", r)
		// if !strings.Contains(fn, "203c") {
		// 	continue
		// }
		sv := NewSVG(math32.Vec2(512, 512))
		upem := float32(1024)
		scale := 82.0 / upem
		_ = scale
		sv.Translate.Y = upem
		// sv.Root.ViewBox.Min.Y = upem
		sv.Root.ViewBox.Size.SetScalar(512)
		// sv.Root.ViewBox.PreserveAspectRatio.Align.Set(svg.AlignNone)
		// sv.Scale = scale
		sv.GroupFilter = fmt.Sprintf("glyph%d", gid)
		sfn := filepath.Join("testdata/font-emoji-src", fn+".svg")
		fmt.Println(sfn, "gid:", sv.GroupFilter, "len:", len(gd.Source))
		b := bytes.NewBuffer(gd.Source)
		err := sv.ReadXML(b)
		assert.NoError(t, err)
		img := sv.RenderImage()
		imfn := filepath.Join("svg", "png", "font-emoji", strings.TrimSuffix(fn, ".svg"))
		imagex.Assert(t, img, imfn)
		// sv.SaveXML(sfn)
		// os.WriteFile(sfn, gd.Source, 0666)
		ctr++
		if ctr > 10 {
			break
		}
	}
}
