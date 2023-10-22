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
	"goki.dev/goosi"
	"goki.dev/icons"
)

func main() { gimain.Run(app) }

func app() {
	goosi.ZoomFactor = 2

	gi.SetAppName("icons")
	gi.SetAppAbout(`This is a demo of the icons in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	// note: can add a path to view other icon sets
	// svg.CurIconSet.OpenIconsFromPath("/Users/oreilly/github/inkscape/share/icons/multicolor/symbolic/actions")

	sc := gi.NewScene("gogi-icons-demo").SetTitle("GoGi Icons")

	grid := gi.NewFrame(sc, "grid").SetLayout(gi.LayoutGrid)
	grid.Style(func(s *styles.Style) {
		s.Columns = int(s.UnContext.Ew / (17 * s.UnContext.FontEm))
		s.AlignH = styles.AlignCenter
		s.Margin.Set(units.Dp(1))
		s.SetStretchMax()
	})

	icnms := icons.All()
	for _, icnm := range icnms {
		ic := strings.TrimSuffix(icnm, ".svg")
		if strings.HasSuffix(ic, "-fill") {
			continue
		}

		vb := gi.NewLayout(grid, ic).SetLayout(gi.LayoutVert)
		vb.Style(func(s *styles.Style) {
			s.MaxWidth = units.Em(15)
		})

		gi.NewLabel(vb, ic).SetText(ic)
		gi.NewIcon(vb, ic).SetIcon(icons.Icon(ic))
	}

	gi.NewWindow(sc).Run().Wait()
}
