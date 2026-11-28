// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

func TestParentActualBackground(t *testing.T) {
	make := func() (b *Body, fr *Frame) {
		b = NewBody()
		fr = NewFrame(b)
		fr.Styler(func(s *styles.Style) {
			s.Min.Set(units.Em(5))
			s.CenterAll()
		})
		NewText(fr).SetType(TextHeadlineSmall).SetText("Test")
		return
	}

	sc, _ := make()
	AssertRender(t, sc, "style/parent-background-color/white")

	sc, fr := make()
	fr.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
	})
	fr.SetState(true, states.Hovered)
	AssertRender(t, sc, "style/parent-background-color/white-hovered-pre")

	sc, fr = make()
	fr.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
	})
	AssertRender(t, sc, "style/parent-background-color/white-hovered-post", func() {
		sc.AsyncLock()
		defer sc.AsyncUnlock()
		fr.SetState(true, states.Hovered)
		fr.StyleTree()
		fr.NeedsRender()
	})

	sc, fr = make()
	fr.Styler(func(s *styles.Style) {
		s.Background = colors.Scheme.OutlineVariant
	})
	AssertRender(t, sc, "style/parent-background-color/gray")

	sc, fr = make()
	fr.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
		s.Background = colors.Scheme.OutlineVariant
	})
	fr.SetState(true, states.Hovered)
	AssertRender(t, sc, "style/parent-background-color/gray-hovered-pre")

	sc, fr = make()
	fr.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
		s.Background = colors.Scheme.OutlineVariant
	})
	AssertRender(t, sc, "style/parent-background-color/gray-hovered-post", func() {
		sc.AsyncLock()
		defer sc.AsyncUnlock()
		fr.SetState(true, states.Hovered)
		fr.StyleTree()
		fr.NeedsRender()
	})
}

func TestParentUnits(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(10), units.Em(20))
	})
	NewWidgetBase(b).Styler(func(s *styles.Style) {
		s.Background = colors.Scheme.Primary.Base
		s.Min.Set(units.Pw(50), units.Ph(75))
	})
	AssertRender(t, b, "style/parent-units")
}

// TestDisplayNoneToggled tests that a widget whose Display is none is
// shown again, with a size, when its Display is no longer none, and that
// a widget made invisible otherwise stays invisible.
func TestDisplayNoneToggled(t *testing.T) {
	b := NewBody()
	hidden := true
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(100), units.Dp(50))
		if hidden {
			s.Display = styles.DisplayNone
		}
	})
	other := NewFrame(b)
	other.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(100), units.Dp(50))
	})
	other.SetState(true, states.Invisible)
	RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.True(t, fr.StateIs(states.Invisible))
		hidden = false
		b.Update()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.False(t, fr.StateIs(states.Invisible), "shown again")
		assert.Greater(t, fr.Geom.Size.Actual.Total.Y, float32(0), "laid out")
		assert.True(t, other.StateIs(states.Invisible), "invisible otherwise")
		hidden = true
		b.Update()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.True(t, fr.StateIs(states.Invisible), "hidden again")
	})
}
