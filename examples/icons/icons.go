// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

func main() {
	b := gi.NewBody("Cogent Core Icons")

	// note: can add a path to view other icon sets
	// svg.CurIconSet.OpenIconsFromPath("/Users/oreilly/github/inkscape/share/icons/multicolor/symbolic/actions")

	grid := gi.NewFrame(b, "grid")
	grid.Style(func(s *styles.Style) {
		s.Wrap = true
		s.Grow.Set(1, 1)
		s.Margin.Set(units.Dp(8))
		s.Overflow.Y = styles.OverflowAuto
	})

	icnms := icons.All()
	for _, ic := range icnms {
		icnm := string(ic)
		if strings.HasSuffix(icnm, "-fill") {
			continue
		}
		vb := gi.NewLayout(grid, icnm).Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Max.X.Em(15) // constraining width exactly gives nice grid-like appearance
			s.Min.X.Em(15)
		})
		gi.NewLabel(vb, icnm).SetText(icnm).Style(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
		gi.NewIcon(vb, icnm).SetIcon(icons.Icon(icnm)).Style(func(s *styles.Style) {
			s.Min.Set(units.Em(4))
		})
	}

	b.RunMainWindow()
}
