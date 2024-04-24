// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

func TestParentActualBackground(t *testing.T) {
	make := func() (b *Body, fr *Frame) {
		b = NewBody()
		fr = NewFrame(b)
		fr.Style(func(s *styles.Style) {
			s.Min.Set(units.Em(5))
			s.CenterAll()
		})
		NewText(fr).SetType(TextHeadlineSmall).SetText("Test")
		return
	}

	sc, _ := make()
	sc.AssertRender(t, "style/parent-background-color/white")

	sc, fr := make()
	fr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
	})
	fr.SetState(true, states.Hovered)
	sc.AssertRender(t, "style/parent-background-color/white-hovered-pre")

	sc, fr = make()
	fr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
	})
	sc.AssertRender(t, "style/parent-background-color/white-hovered-post", func() {
		fr.SetState(true, states.Hovered)
		fr.ApplyStyleTree()
		fr.NeedsRender()
	})

	sc, fr = make()
	fr.Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Scheme.OutlineVariant)
	})
	sc.AssertRender(t, "style/parent-background-color/gray")

	sc, fr = make()
	fr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
		s.Background = colors.C(colors.Scheme.OutlineVariant)
	})
	fr.SetState(true, states.Hovered)
	sc.AssertRender(t, "style/parent-background-color/gray-hovered-pre")

	sc, fr = make()
	fr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
		s.Background = colors.C(colors.Scheme.OutlineVariant)
	})
	sc.AssertRender(t, "style/parent-background-color/gray-hovered-post", func() {
		fr.SetState(true, states.Hovered)
		fr.ApplyStyleTree()
		fr.NeedsRender()
	})
}

func TestParentUnits(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Em(10), units.Em(20))
	})
	NewBox(b).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Scheme.Primary.Base)
		s.Min.Set(units.Pw(50), units.Ph(75))
	})
	b.AssertRender(t, "style/parent-units")
}
