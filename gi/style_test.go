// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"

	"goki.dev/colors"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
)

func TestParentBackgroundColor(t *testing.T) {
	make := func() (sc *Scene, fr *Frame) {
		sc = NewScene()
		fr = NewFrame(sc)
		fr.Style(func(s *styles.Style) {
			s.Min.Set(units.Em(5))
		})
		NewLabel(fr).SetType(LabelTitleLarge).SetText("Test")
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
		fr.ApplyStyle(sc)
		fr.SetNeedsRender()
	})

	sc, fr = make()
	fr.Style(func(s *styles.Style) {
		s.BackgroundColor.SetSolid(colors.Scheme.OutlineVariant)
	})
	sc.AssertPixelsOnShow(t, filepath.Join("style", "parent_background_color", "gray"))

	sc, fr = make()
	fr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
		s.BackgroundColor.SetSolid(colors.Scheme.OutlineVariant)
	})
	fr.SetState(true, states.Hovered)
	sc.AssertPixelsOnShow(t, filepath.Join("style", "parent_background_color", "gray_hovered_pre"))

	sc, fr = make()
	fr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable)
		s.BackgroundColor.SetSolid(colors.Scheme.OutlineVariant)
	})
	sc.AssertPixelsOnShow(t, filepath.Join("style", "parent_background_color", "gray_hovered_post"), func() {
		fr.SetState(true, states.Hovered)
		fr.ApplyStyle(sc)
		fr.SetNeedsRender()
	})
}
