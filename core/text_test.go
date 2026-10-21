// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/base/strcase"
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
