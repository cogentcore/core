// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"github.com/stretchr/testify/assert"
)

// TestStyleDefaults tests the defaults of a style and its derived values.
func TestStyleDefaults(t *testing.T) {
	s := NewStyle()
	assert.Equal(t, Start, s.Align)
	assert.Equal(t, float32(1.3), s.LineHeight)
	assert.Equal(t, float32(1.5), s.ParaSpacing)
	assert.Equal(t, 4, s.TabSize)

	uc := &units.Context{}
	uc.Defaults()
	s.Indent.Dp(10)
	s.ToDots(uc)
	assert.Equal(t, float32(16), s.FontSize.Dots)
	assert.Equal(t, float32(10), s.Indent.Dots)
	rs := rich.NewStyle()
	assert.Equal(t, float32(16), s.FontHeight(rs))
	assert.Equal(t, float32(21), s.LineHeightDots(rs), "ceil of 16 * 1.3")
	rs.Size = 2
	assert.Equal(t, float32(32), s.FontHeight(rs))

	s.SetUnitContext(uc)
	assert.Equal(t, float32(16), uc.FontEm)
	z := &Style{}
	z.SetUnitContext(uc)
	assert.Equal(t, float32(16), uc.FontEm, "a zero size is 16")

	assert.Equal(t, s.Color, s.FillColor(rich.NewStyle()))
	red := color.RGBA{255, 0, 0, 255}
	assert.Equal(t, color.Color(red), s.FillColor(rich.NewStyle().SetFillColor(red)))
}

// TestStyleAlign tests the alignment factors and the white space modes.
func TestStyleAlign(t *testing.T) {
	assert.Equal(t, float32(0), Start.Factor())
	assert.Equal(t, float32(0.5), Center.Factor())
	assert.Equal(t, float32(1), End.Factor())
	assert.Equal(t, float32(0), Justify.Factor())
	s := NewStyle()
	for _, tt := range []struct {
		align, alignV Aligns
		ax, ay        float32
	}{
		{Start, Start, 0, 0.9},
		{Center, Center, 0.5, 0.45},
		{End, End, 1, -0.1},
		{Justify, Justify, 0, 0},
	} {
		s.Align, s.AlignV = tt.align, tt.alignV
		ax, ay := s.AlignFactors()
		assert.Equal(t, tt.ax, ax, tt.align.String())
		assert.Equal(t, tt.ay, ay, tt.alignV.String())
	}

	for _, ws := range []WhiteSpaces{WrapAsNeeded, WrapAlways, WrapSpaceOnly, WhiteSpacePreWrap} {
		assert.True(t, ws.HasWordWrap(), ws.String())
	}
	assert.False(t, WrapNever.HasWordWrap())
	assert.False(t, WhiteSpacePre.HasWordWrap())
	assert.True(t, WhiteSpacePre.KeepWhiteSpace())
	assert.True(t, WhiteSpacePreWrap.KeepWhiteSpace())
	assert.False(t, WrapAsNeeded.KeepWhiteSpace())
}

// TestStyleInherit tests inheriting the fields of a parent style.
func TestStyleInherit(t *testing.T) {
	par := NewStyle()
	par.Align = Center
	par.LineHeight = 2
	par.TabSize = 8
	par.Direction = rich.RTL
	par.WhiteSpace = WhiteSpacePre
	s := &Style{}
	s.InheritFields(par)
	assert.Equal(t, Center, s.Align)
	assert.Equal(t, float32(2), s.LineHeight)
	assert.Equal(t, 8, s.TabSize)
	assert.Equal(t, rich.RTL, s.Direction)
	assert.Equal(t, WrapAsNeeded, s.WhiteSpace, "white space is not inherited")
}

// TestStyleProperties tests setting a style from properties, and getting
// the properties of a style.
func TestStyleProperties(t *testing.T) {
	par := NewStyle()
	par.TabSize = 2
	s := NewStyle()
	s.FromProperties(par, map[string]any{
		"text-align":          "center",
		"text-vertical-align": "end",
		"font-size":           "20px",
		"line-height":         "150%",
		"para-spacing":        2,
		"white-space":         "pre",
		"direction":           "rtl",
		"text-indent":         "2em",
		"tab-size":            "inherit",
		"select-color":        "red",
		"highlight-color":     "#00ff00",
		"#id":                 "ignored",
		"":                    "ignored",
		"unknown":             "ignored",
	}, colors.BaseContext(colors.Black))
	assert.Equal(t, Center, s.Align)
	assert.Equal(t, End, s.AlignV)
	assert.Equal(t, units.Px(20), s.FontSize)
	assert.Equal(t, float32(1.5), s.LineHeight)
	assert.Equal(t, float32(2), s.ParaSpacing)
	assert.Equal(t, WhiteSpacePre, s.WhiteSpace)
	assert.Equal(t, rich.RTL, s.Direction)
	assert.Equal(t, units.Em(2), s.Indent)
	assert.Equal(t, 2, s.TabSize, "inherited")
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, colors.ToUniform(s.SelectColor))
	assert.Equal(t, color.RGBA{0, 255, 0, 255}, colors.ToUniform(s.HighlightColor))

	s.FromProperty(nil, "para-spacing", "initial", nil)
	assert.Equal(t, float32(1.5), s.ParaSpacing, "the default")
	s.FromProperty(par, "select-color", "inherit", nil)
	assert.Equal(t, par.SelectColor, s.SelectColor)
	s.FromProperty(par, "highlight-color", "inherit", nil)
	assert.Equal(t, par.HighlightColor, s.HighlightColor)
	s.FromProperty(nil, "select-color", "initial", nil)
	assert.Equal(t, colors.Scheme.Select.Container, s.SelectColor)
	s.FromProperty(nil, "highlight-color", "initial", nil)
	assert.Equal(t, colors.Scheme.Warn.Container, s.HighlightColor)

	p := map[string]any{}
	s.TabSize = 8
	rs := rich.NewStyle()
	rs.Weight = rich.Bold
	rs.Slant = rich.Italic
	rs.Stretch = rich.Condensed
	rs.Family = rich.Custom
	s.CustomFont = "Mine"
	rs.Decoration.SetFlag(true, rich.Underline)
	rs.SetStrokeColor(colors.Blue)
	s.ToProperties(rs, p)
	assert.Equal(t, "20px", p["font-size"])
	assert.Equal(t, rich.FontName("Mine"), p["font-family"])
	assert.Equal(t, "italic", p["font-style"])
	assert.Equal(t, "bold", p["font-weight"])
	assert.Equal(t, "condensed", p["font-stretch"])
	assert.Contains(t, p["text-decoration"], "underline")
	assert.Equal(t, "center", p["text-align"])
	assert.Equal(t, "end", p["text-vertical-align"])
	assert.Equal(t, "1.5", p["line-height"])
	assert.Equal(t, "Pre", p["white-space"])
	assert.Equal(t, "rtl", p["direction"])
	assert.Equal(t, "8", p["tab-size"])
	assert.Equal(t, "#0000FF", p["stroke-color"])
	assert.Equal(t, colors.AsHex(colors.ToUniform(colors.Scheme.Select.Container)), p["select-color"])
	assert.Equal(t, colors.AsHex(colors.ToUniform(colors.Scheme.Warn.Container)), p["highlight-color"])

	p = map[string]any{}
	NewStyle().ToProperties(rich.NewStyle(), p)
	assert.Equal(t, "none", p["text-decoration"])
	for _, k := range []string{"font-size", "text-align", "line-height", "direction", "tab-size", "font-weight"} {
		assert.NotContains(t, p, k, "the default")
	}
	assert.Contains(t, p, "fill")
}
