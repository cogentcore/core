// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package main

import (
	"cogentcore.org/core/styles/units"
	_ "cogentcore.org/core/system/driver"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped/shapedjs"
	"cogentcore.org/core/text/text"
)

func main() {
	shapedjs.MeasureTest("A")

	sh := shapedjs.NewShaper()
	uc := units.Context{}
	uc.Defaults()
	tx := rich.NewPlainText([]rune("This is a Test"))
	tsty := text.NewStyle()
	tsty.ToDots(&uc)
	runs := sh.Shape(tx, tsty, &rich.DefaultSettings)
	_ = runs
	// lns := sh.WrapLines(tx, fsty, tsty, &rich.DefaultSettings, math32.Vec2(0, 0))
	// fmt.Println("\n\ngo-text\n")
	// fmt.Printf("%#v\n", lns.Lines[0])
	// g := lns.Lines[0].Runs[0].(*shapedgt.Run).Output.Glyphs[0]
	// fmt.Printf("%#v\n", g)
	// fmt.Println("w", math32.FromFixed(g.Width), "h", math32.FromFixed(g.Height), "xb", math32.FromFixed(g.XBearing), "yb", math32.FromFixed(g.YBearing))

	select {}
}
