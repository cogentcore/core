// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"github.com/stretchr/testify/assert"
)

// setProps sets the given properties on a new default style, with the
// given parent (which can be nil).
func setProps(parent *Style, props map[string]any) *Style {
	s := NewStyle()
	for k, v := range props {
		s.FromProperty(parent, k, v, nil)
	}
	return s
}

func TestStyleFromPropertyLayout(t *testing.T) {
	s := setProps(nil, map[string]any{
		"display":          "grid",
		"flex-direction":   "column",
		"flex-grow":        2,
		"wrap":             true,
		"justify-content":  "center",
		"justify-items":    "end",
		"justify-self":     "start",
		"align-content":    "space-between",
		"align-items":      "center",
		"align-self":       "end",
		"x":                "3px",
		"y":                "4px",
		"width":            "10em",
		"height":           "20px",
		"max-width":        "30dp",
		"max-height":       "40dp",
		"margin":           "1dp 2dp",
		"padding":          "3dp",
		"overflow":         "scroll",
		"columns":          3,
		"col-span":         2,
		"scrollbar-width":  "7dp",
		"opacity":          0.5,
		"color":            "red",
		"background-color": "blue",
	})
	assert.Equal(t, Grid, s.Display)
	assert.Equal(t, Column, s.Direction)
	assert.Equal(t, float32(2), s.Grow.Y)
	assert.True(t, s.Wrap)
	assert.Equal(t, Center, s.Justify.Content)
	assert.Equal(t, End, s.Justify.Items)
	assert.Equal(t, Start, s.Justify.Self)
	assert.Equal(t, SpaceBetween, s.Align.Content)
	assert.Equal(t, Center, s.Align.Items)
	assert.Equal(t, End, s.Align.Self)
	assert.Equal(t, units.Px(3), s.Pos.X)
	assert.Equal(t, units.Px(4), s.Pos.Y)
	assert.Equal(t, units.Em(10), s.Min.X, "width sets the min")
	assert.Equal(t, units.Px(20), s.Min.Y)
	assert.Equal(t, units.Dp(30), s.Max.X)
	assert.Equal(t, units.Dp(40), s.Max.Y)
	assert.Equal(t, units.Dp(1), s.Margin.Top)
	assert.Equal(t, units.Dp(2), s.Margin.Right)
	assert.Equal(t, units.Dp(3), s.Padding.Left)
	assert.Equal(t, OverflowScroll, s.Overflow.Y)
	assert.Equal(t, 3, s.Columns)
	assert.Equal(t, 2, s.ColSpan)
	assert.Equal(t, units.Dp(7), s.ScrollbarWidth)
	assert.Equal(t, float32(0.5), s.Opacity)
	assert.Equal(t, colors.Uniform(colors.Red), s.Color)
	assert.Equal(t, colors.Uniform(colors.Blue), s.Background)

	mn := setProps(nil, map[string]any{"min-width": "5dp", "min-height": "6dp"})
	assert.Equal(t, units.Dp(5), mn.Min.X, "min-width is also the min")
	assert.Equal(t, units.Dp(6), mn.Min.Y)

	row := setProps(nil, map[string]any{"flex-direction": "row-reverse"})
	assert.Equal(t, Row, row.Direction)
}

func TestStyleFromPropertyInherit(t *testing.T) {
	parent := setProps(nil, map[string]any{
		"color": "green", "background-color": "yellow", "flex-direction": "column",
		"margin": "4dp", "padding": "5dp", "border-style": "dashed",
		"border-width": "2dp", "border-radius": "3dp", "border-color": "red",
		"font-size": "20dp", "font-family": "Mono", "text-decoration": "underline",
		"select-color": "blue", "highlight-color": "orange",
	})
	keys := []string{"color", "background-color", "flex-direction", "margin", "padding",
		"border-style", "border-width", "border-radius", "border-color",
		"font-size", "font-family", "text-decoration", "select-color", "highlight-color"}
	inh := map[string]any{}
	for _, k := range keys {
		inh[k] = "inherit"
	}
	s := setProps(parent, inh)
	assert.Equal(t, parent.Color, s.Color)
	assert.Equal(t, parent.Background, s.Background)
	assert.Equal(t, Column, s.Direction)
	assert.Equal(t, parent.Margin, s.Margin)
	assert.Equal(t, parent.Padding, s.Padding)
	assert.Equal(t, parent.Border, s.Border)
	assert.Equal(t, parent.Font.Size, s.Font.Size)
	assert.Equal(t, parent.Font.Family, s.Font.Family)
	assert.Equal(t, parent.Font.Decoration, s.Font.Decoration)
	assert.Equal(t, parent.Text.SelectColor, s.Text.SelectColor)
	assert.Equal(t, parent.Text.HighlightColor, s.Text.HighlightColor)

	ini := map[string]any{}
	for _, k := range keys {
		ini[k] = "initial"
	}
	s = setProps(parent, ini)
	assert.Equal(t, colors.Scheme.OnSurface, s.Color)
	assert.Nil(t, s.Background)
	assert.Equal(t, Row, s.Direction)
	assert.Equal(t, float32(0), s.Margin.Top.Value)
	assert.Equal(t, BorderSolid, s.Border.Style.Top)
	assert.Equal(t, units.Dp(16), s.Font.Size)
	assert.Equal(t, rich.SansSerif, s.Font.Family)
	assert.Equal(t, rich.Decorations(0), s.Font.Decoration)
	assert.Equal(t, colors.Scheme.Select.Container, s.Text.SelectColor)
	assert.Equal(t, colors.Scheme.Warn.Container, s.Text.HighlightColor)
}

func TestStyleFromPropertyBorder(t *testing.T) {
	s := setProps(nil, map[string]any{"border-style": BorderDotted})
	assert.Equal(t, BorderDotted, s.Border.Style.Top)
	s = setProps(nil, map[string]any{"border-style": []BorderStyles{BorderDotted, BorderDashed}})
	assert.Equal(t, BorderDashed, s.Border.Style.Right)
	s = setProps(nil, map[string]any{"border-style": int(BorderDouble)})
	assert.Equal(t, BorderDouble, s.Border.Style.Bottom)
	s = setProps(nil, map[string]any{"border-style": "groove", "border-width": "1dp 2dp 3dp 4dp"})
	assert.Equal(t, BorderGroove, s.Border.Style.Left)
	assert.Equal(t, units.Dp(4), s.Border.Width.Left)
	s = setProps(nil, map[string]any{"border-color": "#f00"})
	assert.Equal(t, colors.Uniform(colors.Red), s.Border.Color.Top)
}

// TestStyleOutlineShadowFuncs tests the property functions of outlines
// and shadows, which are not used by [Style.FromProperty].
func TestStyleOutlineShadowFuncs(t *testing.T) {
	parent := &Border{}
	parent.Style.Set(BorderDashed)
	parent.Width.Set(units.Dp(2))
	parent.Radius.Set(units.Dp(3))
	parent.Color.Set(colors.Uniform(colors.Red))
	for key, f := range styleOutlineFuncs {
		var bs Border
		f(&bs, key, "inherit", parent, nil)
		f(&bs, key, "initial", parent, nil)
	}
	var bs Border
	styleOutlineFuncs["outline-style"](&bs, "outline-style", "dotted", nil, nil)
	styleOutlineFuncs["outline-style"](&bs, "outline-style", BorderDouble, nil, nil)
	assert.Equal(t, BorderDouble, bs.Style.Top)
	styleOutlineFuncs["outline-style"](&bs, "outline-style", []BorderStyles{BorderNone}, nil, nil)
	styleOutlineFuncs["outline-style"](&bs, "outline-style", int(BorderGroove), nil, nil)
	assert.Equal(t, BorderGroove, bs.Style.Top)
	styleOutlineFuncs["outline-width"](&bs, "outline-width", "4dp", nil, nil)
	assert.Equal(t, units.Dp(4), bs.Width.Top)
	styleOutlineFuncs["outline-radius"](&bs, "outline-radius", "5dp", nil, nil)
	assert.Equal(t, units.Dp(5), bs.Radius.Top)
	styleOutlineFuncs["outline-color"](&bs, "outline-color", "blue", nil, nil)
	assert.Equal(t, colors.Uniform(colors.Blue), bs.Color.Top)

	psh := &Shadow{Color: colors.Uniform(colors.Red)}
	sh := &Shadow{}
	styleShadowFuncs["box-shadow.color"](sh, "box-shadow.color", "inherit", psh, nil)
	assert.Equal(t, psh.Color, sh.Color)
	styleShadowFuncs["box-shadow.color"](sh, "box-shadow.color", "initial", psh, nil)
	assert.Equal(t, colors.Scheme.Shadow, sh.Color)
	styleShadowFuncs["box-shadow.color"](sh, "box-shadow.color", "green", nil, nil)
	styleShadowFuncs["box-shadow.offset-x"](sh, "box-shadow.offset-x", "2dp", nil, nil)
	styleShadowFuncs["box-shadow.inset"](sh, "box-shadow.inset", true, nil, nil)
	assert.Equal(t, units.Dp(2), sh.OffsetX)
	assert.True(t, sh.Inset)
}

func TestFontTextFromProperties(t *testing.T) {
	var fs Font
	fs.Defaults()
	fs.FromProperties(nil, map[string]any{
		"font-size":       "large",
		"font-family":     "Inter",
		"font-style":      "italic",
		"font-weight":     "bold",
		"font-stretch":    "condensed",
		"text-decoration": "underline",
		"#id":             "ignored",
		"":                "ignored",
	}, nil)
	assert.Equal(t, units.Pt(14), fs.Size)
	assert.Equal(t, rich.Custom, fs.Family)
	assert.Equal(t, rich.FontName("Inter"), fs.CustomFont)
	assert.Equal(t, rich.Italic, fs.Slant)
	assert.Equal(t, rich.Bold, fs.Weight)
	assert.Equal(t, rich.Condensed, fs.Stretch)
	assert.True(t, fs.Decoration.HasFlag(rich.Underline))

	fs.FromProperty(nil, "font-size", "2em", nil)
	assert.Equal(t, units.Em(2), fs.Size)
	fs.FromProperty(nil, "text-decoration", "none", nil)
	assert.Equal(t, rich.Decorations(0), fs.Decoration)
	fs.FromProperty(nil, "text-decoration", rich.LineThrough, nil)
	fs.FromProperty(nil, "text-decoration", int(rich.Overline), nil)
	assert.True(t, fs.Decoration.HasFlag(rich.LineThrough))
	assert.True(t, fs.Decoration.HasFlag(rich.Overline))
	fs.FromProperty(nil, "unknown", 1, nil)

	var ts Text
	ts.Defaults()
	ts.FromProperties(nil, map[string]any{
		"text-align":          "center",
		"text-vertical-align": "end",
		"line-height":         "150%",
		"white-space":         "pre",
		"direction":           "rtl",
		"text-indent":         "2em",
		"text-hanging":        "1em",
		"tab-size":            8,
		"select-color":        "red",
		"highlight-color":     "blue",
		".class":              "ignored",
	}, nil)
	assert.Equal(t, text.Center, ts.Align)
	assert.Equal(t, text.End, ts.AlignV)
	assert.Equal(t, float32(1.5), ts.LineHeight)
	assert.Equal(t, text.WhiteSpacePre, ts.WhiteSpace)
	assert.Equal(t, rich.RTL, ts.Direction)
	assert.Equal(t, units.Em(2), ts.Indent)
	assert.Equal(t, units.Em(1), ts.Hanging)
	assert.Equal(t, 8, ts.TabSize)
	assert.Equal(t, colors.Uniform(colors.Red), ts.SelectColor)
	assert.Equal(t, colors.Uniform(colors.Blue), ts.HighlightColor)
}

func TestPathProperties(t *testing.T) {
	var parent Path
	parent.Defaults()
	parent.Stroke.Color = colors.Uniform(colors.Red)
	parent.Opacity = 0.5

	var pc Path
	pc.Defaults()
	pc.SetProperties(&parent, map[string]any{
		"stroke":            "blue",
		"stroke-opacity":    0.5,
		"stroke-width":      "3px",
		"stroke-dasharray":  "4, 2 1",
		"stroke-linecap":    "round",
		"stroke-linejoin":   "bevel",
		"stroke-miterlimit": 4,
		"fill":              "green",
		"fill-opacity":      0.25,
		"fill-rule":         "evenodd",
		"vector-effect":     "non-scaling-stroke",
		"opacity":           "inherit",
		"transform":         "translate(2,3)",
		"_skip":             1,
	}, nil)
	assert.True(t, pc.StyleSet)
	assert.False(t, pc.PropertiesNil)
	assert.Equal(t, colors.Uniform(colors.Blue), pc.Stroke.Color)
	assert.Equal(t, float32(0.5), pc.Stroke.Opacity)
	assert.Equal(t, units.Px(3), pc.Stroke.Width)
	assert.Equal(t, []float32{4, 2, 1, 4, 2, 1}, pc.Stroke.Dashes, "odd dashes repeat")
	assert.Equal(t, ppath.CapRound, pc.Stroke.Cap)
	assert.Equal(t, ppath.JoinBevel, pc.Stroke.Join)
	assert.Equal(t, float32(4), pc.Stroke.MiterLimit)
	assert.Equal(t, colors.Uniform(colors.Green), pc.Fill.Color)
	assert.Equal(t, float32(0.25), pc.Fill.Opacity)
	assert.Equal(t, ppath.EvenOdd, pc.Fill.Rule)
	assert.Equal(t, ppath.VectorEffectNonScalingStroke, pc.VectorEffect)
	assert.Equal(t, float32(0.5), pc.Opacity, "inherited")
	assert.Equal(t, math32.Translate2D(2, 3), pc.Transform)
	assert.True(t, pc.HasFill())
	pc.ToDotsImpl(units.NewContext())
	assert.True(t, pc.HasStroke())

	props := map[string]any{}
	pc.GetProperties(props)
	assert.Equal(t, map[string]any{
		"stroke": "#0000FF", "stroke-opacity": "0.5", "stroke-width": "3px",
		"stroke-linecap": "round", "stroke-linejoin": "bevel",
		"fill": "#008000", "fill-opacity": "0.25", "fill-rule": "evenodd",
	}, props)

	// the colors, dashes, and transform of other kinds and inheritance
	pc.SetProperties(&parent, map[string]any{"stroke": "inherit", "fill": "initial",
		"stroke-dasharray": []float32{1, 2}, "transform": math32.Scale2D(2, 2)}, nil)
	assert.Equal(t, parent.Stroke.Color, pc.Stroke.Color)
	assert.Equal(t, colors.Uniform(colors.Black), pc.Fill.Color)
	assert.Equal(t, []float32{1, 2}, pc.Stroke.Dashes)
	assert.Equal(t, math32.Scale2D(2, 2), pc.Transform)
	m := math32.Rotate2D(1)
	ds := []float32{3, 4}
	pc.SetProperties(nil, map[string]any{"stroke-dasharray": &ds, "transform": &m, "stroke": "not a color", "fill": "not a color"}, nil)
	assert.Equal(t, []float32{3, 4}, pc.Stroke.Dashes)
	assert.Equal(t, m, pc.Transform)
	assert.Equal(t, colors.Uniform(colors.Black), pc.Fill.Color, "an invalid color is ignored")
	pc.SetProperties(&parent, map[string]any{"stroke-dasharray": "initial", "transform": "initial"}, nil)
	assert.Nil(t, pc.Stroke.Dashes)
	assert.Equal(t, math32.Identity2(), pc.Transform)
	pc.SetProperties(&parent, map[string]any{"stroke-dasharray": "inherit", "transform": "inherit"}, nil)
	assert.Equal(t, parent.Transform, pc.Transform)

	pc.SetProperties(nil, map[string]any{"display": "none"}, nil)
	assert.False(t, pc.Display)
	props = map[string]any{}
	pc.GetProperties(props)
	assert.Equal(t, map[string]any{"display": "none"}, props)
	pc.SetProperties(nil, map[string]any{"display": "inline"}, nil)
	assert.True(t, pc.Display)
	pc.SetProperties(nil, map[string]any{"display": "block"}, nil)
	assert.True(t, pc.Display)
	pc.SetProperties(nil, nil, nil)
	assert.True(t, pc.PropertiesNil)

	pc.Stroke.Color, pc.Fill.Color = nil, nil
	props = map[string]any{}
	pc.GetProperties(props)
	assert.Equal(t, map[string]any{"stroke": "none", "fill": "none"}, props)
	assert.False(t, pc.HasFill())
	assert.False(t, pc.HasStroke())
}

func TestParseDashesString(t *testing.T) {
	assert.Nil(t, parseDashesString(""))
	assert.Nil(t, parseDashesString("none"))
	assert.Equal(t, []float32{1, 2}, parseDashesString("1px 2px"))
	assert.Nil(t, parseDashesString("1 x"))
}

func TestPaint(t *testing.T) {
	parent := NewPaint()
	parent.Font.Weight = rich.Bold
	pc := NewPaintWithContext(units.NewContext())
	pc.SetProperties(parent, map[string]any{"fill": "red", "#x": 1, "": 2}, nil)
	assert.Equal(t, rich.Bold, pc.Font.Weight, "inherited the first time")
	assert.Equal(t, colors.Uniform(colors.Red), pc.Fill.Color)

	cp := NewPaint()
	cp.CopyStyleFrom(pc)
	assert.Equal(t, pc.Fill, cp.Fill)
	assert.Equal(t, pc.Font, cp.Font)

	props := map[string]any{}
	pc.GetProperties(props)
	assert.Equal(t, "#FF0000", props["fill"])

	st := NewStyle()
	st.Font.Weight = rich.Black
	pc.FromStyle(st)
	assert.Equal(t, rich.Black, pc.Font.Weight)

	pc.Stroke.Width = units.Vw(1)
	pc.SetUnitContextExt(image.Pt(500, 100))
	assert.Equal(t, float32(5), pc.Stroke.Width.Dots)
	pc.PropertiesNil = true
	pc.ToDots()
	pc.Stroke.Width.Value = 2
	pc.ToDots()
	assert.Equal(t, float32(5), pc.Stroke.Width.Dots, "cached for the same context and no properties")
	pc.PropertiesNil = false
	pc.ToDots()
	assert.Equal(t, float32(10), pc.Stroke.Width.Dots)

	var p Path
	p.Defaults()
	p.FromStyle(st)
	assert.Equal(t, st.UnitContext, p.UnitContext)
	var p2 Path
	p2.CopyStyleFrom(&p)
	assert.Equal(t, p.Stroke, p2.Stroke)
}

func TestStrokeApplyBorderStyle(t *testing.T) {
	var ss Stroke
	ss.Defaults()
	ss.Color = colors.Uniform(colors.Red)
	ss.ApplyBorderStyle(BorderDotted)
	assert.Equal(t, []float32{0, 12}, ss.Dashes)
	assert.Equal(t, ppath.CapRound, ss.Cap)
	ss.ApplyBorderStyle(BorderDashed)
	assert.Equal(t, []float32{8, 6}, ss.Dashes)
	ss.ApplyBorderStyle(BorderSolid)
	assert.NotNil(t, ss.Color)
	ss.ApplyBorderStyle(BorderNone)
	assert.Nil(t, ss.Color)
}

func TestRichText(t *testing.T) {
	s := NewStyle()
	s.Font.Weight = rich.Bold
	s.Font.CustomFont = "Inter"
	s.Font.Size = units.Dp(20)
	s.Text.Align = text.Center
	s.Text.TabSize = 2
	s.Color = colors.Uniform(colors.Red)
	s.Opacity = 0.5
	sty, tsty := s.NewRichText()
	assert.Equal(t, rich.Bold, sty.Weight)
	assert.Equal(t, text.Center, tsty.Align)
	assert.Equal(t, 2, tsty.TabSize)
	assert.Equal(t, units.Dp(20), tsty.FontSize)
	assert.Equal(t, rich.FontName("Inter"), tsty.CustomFont)
	assert.Equal(t, colors.ApplyOpacity(color.Color(colors.Red), 0.5), tsty.Color)

	var font Font
	var txt Text
	SetFromRichText(sty, tsty, &font, &txt)
	assert.Equal(t, rich.Bold, font.Weight)
	assert.Equal(t, units.Dp(20), font.Size)
	assert.Equal(t, text.Center, txt.Align)
	assert.Equal(t, 2, txt.TabSize)
}
