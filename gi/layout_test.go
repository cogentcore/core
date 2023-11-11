// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build offscreen

package gi

import (
	"image"
	"os"
	"testing"

	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/driver"
	"goki.dev/mat32/v2"
)

// LayoutFlex tests the core layout flex logic
// See this spreadsheet for the core logic applied to the Grid case:
// https://docs.google.com/spreadsheets/d/1eimUOIJLyj60so94qUr4Buzruj2ulpG5o6QwG2nyxRw/edit?usp=sharing
func TestLayoutFlex(t *testing.T) {
	driver.Main(func(app goosi.App) {
		LayoutTrace = true

		Init()
		sc := NewScene("testflex")
		sc.SceneGeom.Size = image.Point{600, 400}

		sc.Style(func(s *styles.Style) {
			s.Display = styles.DisplayGrid
			s.Columns = 2
			s.MainAxis = mat32.X
			s.Gap.Set(units.Dot(4))
		})

		sp1 := NewSpace(sc, "sp1")
		sp1.Style(func(s *styles.Style) {
			s.Min.X.Dot(10)
			s.Min.Y.Dot(5)
			s.Padding.Set(units.Dot(2))
			s.Margin.Set(units.Dot(3))
			s.Grow.Set(2, 1)
		})

		sp2 := NewSpace(sc, "sp2")
		sp2.Style(func(s *styles.Style) {
			s.Min.X.Dot(25)
			s.Min.Y.Dot(10)
			s.Grow.Set(1, 1)
		})

		sp3 := NewSpace(sc, "sp3")
		sp3.Style(func(s *styles.Style) {
			s.Min.X.Dot(5)
			s.Min.Y.Dot(2)
			s.Grow.Set(2, 1)
		})

		sp4 := NewSpace(sc, "sp4")
		sp4.Style(func(s *styles.Style) {
			s.Min.X.Dot(15)
			s.Min.Y.Dot(15)
			s.Grow.Set(1, 1)
		})

		sc.ConfigScene()
		sc.ApplyStyleScene()
		sc.LayoutScene()

		// app.Quit() // todo: doesn't work
		os.Exit(0)
	})
}
