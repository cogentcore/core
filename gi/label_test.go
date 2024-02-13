// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

func TestLabel(t *testing.T) {
	for _, str := range testStrings {
		if str == "" {
			continue
		}
		for _, typ := range LabelTypesValues() {
			b := NewBody()
			NewLabel(b).SetType(typ).SetText(str)
			b.AssertRender(t, testName("label", str, typ))
		}

		b := NewBody()
		NewLabel(b).SetText(str).Style(func(s *styles.Style) {
			s.Font.Size = units.Rem(2)
		})
		b.AssertRender(t, testName("label", str, "rem"))
	}
}

func TestLabelTextDecoration(t *testing.T) {
	for d := styles.Underline; d <= styles.LineThrough; d++ {
		for st := styles.FontNormal; st <= styles.Italic; st++ {
			d := d
			st := st
			b := NewBody()
			NewLabel(b).SetText("Test").Style(func(s *styles.Style) {
				s.Font.SetDecoration(d)
				s.Font.Style = st
			})
			b.AssertRender(t, "label/text-decoration/"+d.BitIndexString()+"-"+st.String())
		}
	}
}
