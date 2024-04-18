// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

func TestLabelTypes(t *testing.T) {
	for _, typ := range LabelTypesValues() {
		b := NewBody()
		NewLabel(b).SetType(typ).SetText("Hello, world!")
		b.AssertRender(t, "label/type/"+strcase.ToKebab(typ.String()))
	}
}

func TestLabelRem(t *testing.T) {
	b := NewBody()
	NewLabel(b).SetText("Hello, world!").Style(func(s *styles.Style) {
		s.Font.Size = units.Rem(2)
	})
	b.AssertRender(t, "label/rem")
}

func TestLabelTextDecoration(t *testing.T) {
	for d := styles.Underline; d <= styles.LineThrough; d++ {
		for st := styles.FontNormal; st <= styles.Italic; st++ {
			b := NewBody()
			NewLabel(b).SetText("Test").Style(func(s *styles.Style) {
				s.Font.SetDecoration(d)
				s.Font.Style = st
			})
			b.AssertRender(t, "label/text-decoration/"+d.BitIndexString()+"-"+st.String())
		}
	}
}
