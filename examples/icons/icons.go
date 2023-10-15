// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"path/filepath"
	"strings"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/glop/dirs"
	"goki.dev/goosi"
	"goki.dev/icons"
)

func main() { gimain.Run(app) }

func app() {
	nColumns := 5

	goosi.ZoomFactor = 2

	gi.SetAppName("icons")
	gi.SetAppAbout(`This is a demo of the icons in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	// note: can add a path to view other icon sets
	// svg.CurIconSet.OpenIconsFromPath("/Users/oreilly/github/inkscape/share/icons/multicolor/symbolic/actions")

	sc := gi.NewScene("gogi-icons-demo").SetTitle("GoGi Icons")

	grid := gi.NewFrame(sc, "grid").SetLayout(gi.LayoutGrid)
	grid.Style(func(s *styles.Style) {
		// grid.Stripes = gi.RowStripes
		s.Columns = nColumns
		s.AlignH = styles.AlignCenter
		s.Margin.Set(units.Dp(1))
		s.SetStretchMax()
	})

	// il := gi.TheIconMgr.IconList(true)

	icnms := dirs.ExtFileNamesFS(icons.Icons, "svg", ".svg")
	for _, icnm := range icnms {
		// if icnm.IsNil() || strings.HasSuffix(icnms, "-fill") {
		// 	continue
		// }
		// fmt.Println(icnm)
		vb := gi.NewLayout(grid, "vb").SetLayout(gi.LayoutVert)
		vb.Style(func(s *styles.Style) {
			s.MaxWidth = units.Em(15)
			// s.Overflow = OverflowHidden
		})
		gi.NewLabel(vb, "lab1").SetText(strings.TrimSuffix(icnm, ".svg"))

		smico := gi.NewIcon(vb, icnm)
		smico.SetIcon(icons.Icon(filepath.Join("svg", icnm)))
		smico.Style(func(s *styles.Style) {
			s.SetMinPrefWidth(units.Dp(24))
			s.SetMinPrefHeight(units.Dp(24))
			s.BackgroundColor.SetSolid(colors.Transparent)
			// s.Setico.SetProp("fill", colors.Scheme.OnBackground)
			s.Color = colors.Scheme.OnBackground
		})
	}

	gi.NewWindow(sc).Run().Wait()
}
