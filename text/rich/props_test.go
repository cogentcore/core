// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"github.com/stretchr/testify/assert"
)

// TestFromProperties tests setting a style from properties, including
// inherit and initial.
func TestFromProperties(t *testing.T) {
	parent := NewStyle()
	parent.Size = 2
	parent.SetFillColor(colors.Red)
	parent.SetDecoration(Overline)

	s := NewStyle()
	s.FromProperties(parent, map[string]any{
		"font-size":        "x-large",
		"font-family":      "monospace",
		"font-style":       "italic",
		"font-weight":      "bold",
		"font-stretch":     "condensed",
		"text-decoration":  "underline",
		"direction":        "rtl",
		"color":            "blue",
		"stroke-color":     "green",
		"background-color": "yellow",
		"#id":              "ignored",
		"":                 "ignored",
	}, nil)
	assert.Equal(t, FontSizes["x-large"], s.Size)
	assert.Equal(t, Monospace, s.Family)
	assert.Equal(t, Italic, s.Slant)
	assert.Equal(t, Bold, s.Weight)
	assert.Equal(t, Condensed, s.Stretch)
	assert.True(t, s.Decoration.HasFlag(Underline))
	assert.Equal(t, RTL, s.Direction)
	assert.Equal(t, colors.AsRGBA(colors.Blue), colors.AsRGBA(s.FillColor()))
	assert.Equal(t, colors.AsRGBA(colors.Green), colors.AsRGBA(s.StrokeColor()))
	assert.Equal(t, colors.AsRGBA(colors.Yellow), colors.AsRGBA(s.Background()))

	s.FromProperty(parent, "opacity", 0.5, nil)
	assert.InDelta(t, 128, colors.AsRGBA(s.FillColor()).A, 1)
	assert.InDelta(t, 128, colors.AsRGBA(s.Background()).A, 1)

	s.FromProperty(parent, "font-size", "50%", nil)
	assert.Equal(t, float32(0.5), s.Size)
	s.FromProperty(parent, "font-size", "1.5em", nil)
	assert.Equal(t, float32(1.5), s.Size)
	s.FromProperty(parent, "font-size", "inherit", nil)
	assert.Equal(t, float32(2), s.Size)
	s.FromProperty(parent, "font-size", "initial", nil)
	assert.Equal(t, float32(1), s.Size)

	s.FromProperty(parent, "text-decoration", "inherit", nil)
	assert.True(t, s.Decoration.HasFlag(Overline))
	s.FromProperty(parent, "text-decoration", "none", nil)
	assert.False(t, s.Decoration.HasFlag(Underline))
	assert.NotNil(t, s.FillColor(), "none keeps the colors")
	var deco Decorations // a set of flags, not one flag index
	deco.SetFlag(true, LineThrough)
	s.FromProperty(parent, "text-decoration", deco, nil)
	assert.True(t, s.Decoration.HasFlag(LineThrough))

	s.FromProperty(parent, "color", "inherit", nil)
	assert.Equal(t, colors.AsRGBA(colors.Red), colors.AsRGBA(s.FillColor()))
	s.FromProperty(parent, "color", "initial", nil)
	assert.Nil(t, s.FillColor())
	s.FromProperty(parent, "color", "currentColor", nil)
	assert.Equal(t, colors.AsRGBA(colors.Red), colors.AsRGBA(s.FillColor()))
	s.FromProperty(nil, "color", "currentcolor", nil)
	assert.Nil(t, s.FillColor())
	s.FromProperty(parent, "stroke-color", "initial", nil)
	assert.Nil(t, s.StrokeColor())
	s.FromProperty(parent, "background-color", "inherit", nil)
	assert.Nil(t, s.Background(), "the parent has no background")
	s.FromProperty(parent, "unknown-property", 1, nil)
}

// TestSetFromHTMLTag tests the styles of the simple HTML tags.
func TestSetFromHTMLTag(t *testing.T) {
	tags := map[string]func(s *Style) bool{
		"strong": func(s *Style) bool { return s.Weight == Bold },
		"em":     func(s *Style) bool { return s.Slant == Italic },
		"ins":    func(s *Style) bool { return s.Decoration.HasFlag(Underline) },
		"del":    func(s *Style) bool { return s.Decoration.HasFlag(LineThrough) },
		"small":  func(s *Style) bool { return s.Size == 0.8 },
		"big":    func(s *Style) bool { return s.Size == 1.2 },
		"large":  func(s *Style) bool { return s.Size == FontSizes["large"] },
		"mark":   func(s *Style) bool { return s.Background() != nil },
		"abbr":   func(s *Style) bool { return s.Decoration.HasFlag(DottedUnderline) },
		"code":   func(s *Style) bool { return s.Family == Monospace && s.Background() != nil },
	}
	for tag, check := range tags {
		s := NewStyle()
		assert.True(t, s.SetFromHTMLTag(tag), tag)
		assert.True(t, check(s), tag)
	}
	assert.False(t, NewStyle().SetFromHTMLTag("div"))
	assert.Equal(t, "b", Bold.HTMLTag())
	assert.Equal(t, "", Normal.HTMLTag())
}

// TestStyleAccessors tests the smaller accessors and setters of styles.
func TestStyleAccessors(t *testing.T) {
	parent := NewStyle()
	parent.Family = Serif
	parent.Slant = Italic
	parent.Size = 3
	parent.Weight = Bold
	s := NewStyle()
	s.InheritFields(parent)
	assert.Equal(t, Serif, s.Family)
	assert.Equal(t, Italic, s.Slant)
	assert.Equal(t, float32(3), s.Size)
	assert.Equal(t, Bold, s.Weight)
	assert.Contains(t, s.FontFamily(&Settings), "serif")

	assert.Equal(t, float32(700), Bold.ToFloat32())
	assert.Equal(t, float32(1), StretchNormal.ToFloat32())
	assert.True(t, TTB.IsVertical())
	assert.False(t, LTR.IsVertical())
	assert.Equal(t, RTL, Directions(LTR.ToGoText()+1)) // the same numbering

	s.SetLink("https://example.com")
	assert.Equal(t, Link, s.Special)
	assert.Equal(t, "https://example.com", s.URL)
	assert.True(t, s.Decoration.HasFlag(Underline))
	assert.NotNil(t, s.FillColor())
	assert.False(t, s.IsMath())
	s.Special = MathDisplay
	assert.True(t, s.IsMath())
	s.SetStrokeColor(color.RGBA{1, 2, 3, 255})
	assert.Equal(t, color.RGBA{1, 2, 3, 255}, s.StrokeColor())
	s.SetStrokeColor(nil)
	assert.Nil(t, s.StrokeColor())
}

// TestSettingsFamilies tests the font family settings.
func TestSettingsFamilies(t *testing.T) {
	assert.Equal(t, "serif", AddFamily("", "serif"))
	assert.Equal(t, "Times, serif", AddFamily("Times", "serif"))
	assert.Equal(t, []string{"a", "b c", "d"}, FamiliesToList(" a, b c,,d "))
	var st SettingsData
	st.Defaults()
	for fam := range Custom {
		assert.NotEmpty(t, st.Family(fam), fam.String())
	}
	st.Monospace = "Go Mono"
	assert.Equal(t, "Go Mono, monospace, emoji", st.Family(Monospace))
}

// TestTextEditing tests the functions that add, insert, and change the
// spans of texts.
func TestTextEditing(t *testing.T) {
	plain := NewStyle()
	bold := NewStyle().SetWeight(Bold)
	var tx Text
	tx.AddRunes([]rune("Hi "))
	assert.Equal(t, 1, tx.NumSpans(), "a default span is made")
	tx.AddRunes([]rune("there"))
	assert.Equal(t, "Hi there", string(tx.Join()))
	tx.InsertSpan(0, bold, []rune("Oh, "))
	assert.Equal(t, "Oh, Hi there", string(tx.Join()))
	tx.SetSpanRunes(0, []rune("Ah, "))
	sty, r := tx.Span(0)
	assert.Equal(t, Bold, sty.Weight)
	assert.Equal(t, "Ah, ", string(r))

	sp := tx.Split()
	assert.Equal(t, [][]rune{[]rune("Ah, "), []rune("Hi there")}, sp)
	cp := tx.SplitCopy()
	cp[0][0] = 'X'
	assert.Equal(t, 'A', tx.At(0), "a copy")

	tx.AddSuper(plain, "2")
	tx.AddSub(plain, "i")
	tx.AddMathInline(plain, "x")
	tx.AddMathDisplay(plain, "y")
	specials := []Specials{}
	for i := range tx.NumSpans() {
		s, _ := tx.Span(i)
		specials = append(specials, s.Special)
	}
	assert.Equal(t, []Specials{Nothing, Nothing, Super, End, Sub, End, MathInline, End, MathDisplay, End}, specials)

	tx.InsertSpan(2, plain, []rune("!"))
	tx.InsertEndSpecial(3)
	s, _ := tx.Span(3)
	assert.Equal(t, End, s.Special)

	cl := tx.Clone()
	cl.SetSpanRunes(0, []rune("Changed"))
	_, r = tx.Span(0)
	assert.Equal(t, "Ah, ", string(r), "the clone is separate")

	j := Join(NewPlainText([]rune("a")), NewText(bold, []rune("b")))
	assert.Equal(t, 2, j.NumSpans())
	assert.Equal(t, "ab", string(j.Join()))
	j.DebugDump()
}
