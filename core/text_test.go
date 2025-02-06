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
		s.Text.FontSize = units.Rem(2)
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
