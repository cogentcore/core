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
	"goki.dev/goosi"
	"goki.dev/goosi/driver"
	"goki.dev/mat32/v2"
)

func TestLayoutFlex(t *testing.T) {
	driver.Main(func(app goosi.App) {
		LayoutTrace = true

		Init()
		sc := NewScene("testflex")
		sc.Geom.Size = image.Point{600, 400}

		sc.Style(func(s *styles.Style) {
			// s.Display = styles.DisplayGrid
			s.Columns = 2
			s.MainAxis = mat32.X
		})

		sp1 := NewSpace(sc, "sp1")
		sp1.Style(func(s *styles.Style) {
			s.Min.X.Dp(10)
			s.Min.Y.Dp(5)
			s.Grow.Set(2, 1)
		})

		sp2 := NewSpace(sc, "sp2")
		sp2.Style(func(s *styles.Style) {
			s.Min.X.Dp(20)
			s.Min.Y.Dp(10)
			s.Grow.Set(1, 1)
		})

		sp3 := NewSpace(sc, "sp3")
		sp3.Style(func(s *styles.Style) {
			s.Min.X.Dp(5)
			s.Min.Y.Dp(2)
			s.Grow.Set(2, 1)
		})

		sp4 := NewSpace(sc, "sp4")
		sp4.Style(func(s *styles.Style) {
			s.Min.X.Dp(15)
			s.Min.Y.Dp(15)
			s.Grow.Set(1, 1)
		})

		sc.ConfigScene()
		sc.ApplyStyleScene()
		sc.LayoutScene()

		// app.Quit() // todo: doesn't work
		os.Exit(0)
	})
}
