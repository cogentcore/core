// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
)

func TestParentBackgroundColor(t *testing.T) {
	make := func() (sc *Scene, fr *Frame) {
		sc = NewScene()
		fr = NewFrame(sc)
		fr.Style(func(s *styles.Style) {
			s.Min.Set(units.Em(20))
		})
		NewLabel(fr).SetText("Test")
		return
	}

	sc, _ := make()
	sc.AssertPixelsOnShow(t, filepath.Join("style", "parent_background_color_base"))

	sc, fr := make()
	fr.Style(func(s *styles.Style) {
		s.BackgroundColor.SetSolid(colors.Scheme.OutlineVariant)
	})
	sc.AssertPixelsOnShow(t, filepath.Join("style", "parent_background_color_gray"))
}
