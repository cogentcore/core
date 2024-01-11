// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"

	"goki.dev/goki/gi"
	"goki.dev/goki/styles"
	"goki.dev/goki/units"
	"goki.dev/icons"
)

func main() {
	b := gi.NewAppBody("Goki Icons")
	b.App().About = `This is a demo of the icons in the <b>GoGi</b> graphical interface system, within the <b>Goki</b> tree framework.  See <a href="https://github.com/goki">Goki on GitHub</a>`

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

	b.NewWindow().Run().Wait()
}
