// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/icons"
)

func main() { gimain.Run(app) }

func app() {
	gi.SetAppName("icons")
	gi.SetAppAbout(`This is a demo of the icons in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	// note: can add a path to view other icon sets
	// svg.CurIconSet.OpenIconsFromPath("/Users/oreilly/github/inkscape/share/icons/multicolor/symbolic/actions")

	sc := gi.NewScene("gogi-icons-demo").SetTitle("GoGi Icons")

	sc.Style(func(s *styles.Style) {
		s.Align.X = styles.AlignStart
	})

	grid := gi.NewFrame(sc, "grid")
	grid.Style(func(s *styles.Style) {
		// s.Display = styles.Grid
		// s.Columns = 4 // int(s.UnContext.Elw / (17 * s.UnContext.FontEm))
		s.Wrap = true
		s.Grow.Set(1, 1)
		s.Align.X = styles.AlignStart
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

	gi.NewWindow(sc).Run().Wait()
}
