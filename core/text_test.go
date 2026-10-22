// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image/color"
	"testing"

	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
)

func TestTextTypes(t *testing.T) {
	for _, typ := range TextTypesValues() {
		b := NewBody()
		NewText(b).SetType(typ).SetText("Hello, world!")
		b.AssertRender(t, "text/type/"+strcase.ToKebab(typ.String()))
	}
}

func TestTextRem(t *testing.T) {
	b := NewBody()
	NewText(b).SetText("Hello, world!").Styler(func(s *styles.Style) {
		s.Font.Size = units.Rem(2)
	})
	b.AssertRender(t, "text/rem")
}

func TestTextDecoration(t *testing.T) {
	for d := rich.Underline; d <= rich.LineThrough; d++ {
		for st := rich.SlantNormal; st <= rich.Italic; st++ {
			b := NewBody()
			NewText(b).SetText("Test").Styler(func(s *styles.Style) {
				s.Font.SetDecoration(d)
				s.Font.Slant = st
			})
			b.AssertRender(t, "text/decoration/"+d.BitIndexString()+"-"+st.String())
		}
	}
}

func TestTextLink(t *testing.T) {
	b := NewBody()
	NewText(b).SetText(`Hello, <a href="https://example.com">world</a>!`)
	b.AssertRender(t, "text/link")
}

// A restyle that changes neither the text nor its style keeps the rich
// text, and one that changes either makes it again.
func TestTextRichTextSource(t *testing.T) {
	b := NewBody()
	tx := NewText(b).SetText(`Some <b>bold</b> and <a href="https://example.com">linked</a> text`)
	bold := false
	tx.Styler(func(s *styles.Style) {
		if bold {
			s.Font.Weight = rich.Bold
		}
	})
	tx.Style()
	first := tx.richText
	if len(first) < 3 || len(tx.Links) != 1 {
		t.Fatalf("rich text %v with links %v", first, tx.Links)
	}
	tx.Style()
	if &tx.richText[0][0] != &first[0][0] {
		t.Error("restyle made the rich text again")
	}

	bold = true
	tx.Style()
	if &tx.richText[0][0] == &first[0][0] || tx.richText.String() == first.String() {
		t.Error("style change kept the rich text")
	}
	styled := tx.richText

	tx.SetText("Other <i>text</i>")
	tx.Style()
	if &tx.richText[0][0] == &styled[0][0] || len(tx.Links) != 0 {
		t.Error("text change kept the rich text")
	}
}

// linkAndCodeColors returns the color of the link and the background
// of the code in the rich text of the given text.
func linkAndCodeColors(t *testing.T, tx *Text) (link, code color.RGBA) {
	t.Helper()
	for si := range tx.richText {
		s, _ := tx.richText.Span(si)
		switch {
		case s.Special == rich.Link:
			link = colors.AsRGBA(s.FillColor())
		case s.Family == rich.Monospace:
			code = colors.AsRGBA(s.Background())
		}
	}
	if link == (color.RGBA{}) || code == (color.RGBA{}) {
		t.Fatalf("no link or code colors in %v", tx.richText)
	}
	return
}

// A change of the color scheme makes the rich text again, so that the
// colors of links and code follow it.
func TestTextRichTextScheme(t *testing.T) {
	palette, schemes, scheme, dark := colors.Palette, colors.Schemes, colors.Scheme, matcolor.SchemeIsDark
	defer func() {
		colors.Palette, colors.Schemes, colors.Scheme, matcolor.SchemeIsDark = palette, schemes, scheme, dark
	}()
	colors.SetScheme(false)

	b := NewBody()
	tx := NewText(b).SetText(`A <a href="https://example.com">link</a> and <code>code</code>`)
	tx.Style()
	link, code := linkAndCodeColors(t, tx)
	if link != colors.ToUniform(colors.Scheme.Primary.Base) {
		t.Fatalf("link color %v is not the primary color %v", link, colors.Scheme.Primary.Base)
	}

	colors.SetSchemes(color.RGBA{200, 30, 30, 255})
	tx.Style()
	red, _ := linkAndCodeColors(t, tx)
	if red == link || red != colors.ToUniform(colors.Scheme.Primary.Base) {
		t.Errorf("link color %v after a primary color change, was %v", red, link)
	}

	colors.SetScheme(true)
	tx.Style()
	darkLink, darkCode := linkAndCodeColors(t, tx)
	if darkLink == red || darkCode == code || darkLink != colors.ToUniform(colors.Scheme.Primary.Base) {
		t.Errorf("link and code colors %v, %v after a dark mode change, were %v, %v", darkLink, darkCode, red, code)
	}
}

// A full rebuild ([UpdateAll]) makes the rich text again, even for a
// change of the color scheme in place.
func TestTextRichTextRebuild(t *testing.T) {
	b := NewBody()
	tx := NewText(b).SetText(`A <a href="https://example.com">link</a> and <code>code</code>`)
	b.runAndShowNewWindow()
	rw := b.Scene.RenderWindow()
	b.waitNoEvents(rw)

	b.AsyncLock()
	scheme := colors.Scheme // the window can set the scheme when it is shown
	primary := scheme.Primary.Base
	defer func() { scheme.Primary.Base = primary }()
	scheme.Primary.Base = colors.Uniform(color.RGBA{200, 30, 30, 255})
	b.AsyncUnlock()
	UpdateAll()
	rw.waitRenderDone()
	rw.renderWindow()
	b.AsyncLock()
	rebuilt, _ := linkAndCodeColors(t, tx)
	b.AsyncUnlock()
	if rebuilt != colors.ToUniform(scheme.Primary.Base) {
		t.Errorf("link color %v after a rebuild is not %v", rebuilt, scheme.Primary.Base)
	}

	rc := b.Scene.renderContext()
	rc.Lock()
	b.Close()
	rc.Unlock()
}
