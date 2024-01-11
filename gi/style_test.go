// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"

	"goki.dev/abilities"
	"goki.dev/colors"
	"goki.dev/states"
	"goki.dev/styles"
	"goki.dev/units"
)

func TestParentActualBackground(t *testing.T) {
	make := func() (sc *Scene, fr *Frame) {
		sc = NewScene()
		fr = NewFrame(sc)
		fr.Style(func(s *styles.Style) {
			s.Min.Set(units.Em(5))
			s.Align.Content = styles.Center
			s.Justify.Content = styles.Center
		})
		NewLabel(fr).SetType(LabelHeadlineSmall).SetText("Test")
		return
	}

	sc, _ := make()
	sc.AssertPixelsOnShow(t, filepath.Join("style", "parent_background_color", "white"))

	sc, fr := make()
	fr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
	})
	fr.SetState(true, states.Hovered)
	sc.AssertPixelsOnShow(t, filepath.Join("style", "parent_background_color", "white_hovered_pre"))

	sc, fr = make()
	fr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
	})
	sc.AssertPixelsOnShow(t, filepath.Join("style", "parent_background_color", "white_hovered_post"), func() {
		fr.SetState(true, states.Hovered)
		fr.ApplyStyleTree()
		fr.SetNeedsRender(true)
	})

	sc, fr = make()
	fr.Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Scheme.OutlineVariant)
	})
	sc.AssertPixelsOnShow(t, filepath.Join("style", "parent_background_color", "gray"))

	sc, fr = make()
	fr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
		s.Background = colors.C(colors.Scheme.OutlineVariant)
	})
	fr.SetState(true, states.Hovered)
	sc.AssertPixelsOnShow(t, filepath.Join("style", "parent_background_color", "gray_hovered_pre"))

	sc, fr = make()
	fr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
		s.Background = colors.C(colors.Scheme.OutlineVariant)
	})
	sc.AssertPixelsOnShow(t, filepath.Join("style", "parent_background_color", "gray_hovered_post"), func() {
		fr.SetState(true, states.Hovered)
		fr.ApplyStyleTree()
		fr.SetNeedsRender(true)
	})
}
