// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint_test

import (
	"fmt"
	"image"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	. "cogentcore.org/core/paint"
	"cogentcore.org/core/text/htmltext"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

func TestTextAscii(t *testing.T) {
	size := image.Point{300, 100}
	sizef := math32.FromPoint(size)
	txtSh := shaped.NewShaper()
	lines := []string{
		"`0123456789-=~!@#$%^&*()_+",
		"[]\\;',./{}|:\"<>?",
		"ABCDEFGHIJKLMNOPQRSTUV",
		"abcdefghijklmnopqrstuvwxyz",
	}
	for fam := rich.SansSerif; fam < rich.FamilyN; fam++ {
		// if fam != rich.Monospace {
		// 	continue
		// }
		fnm := fam.String()
		RunTest(t, "text/ascii-"+fnm, size.X, size.Y, func(pc *Painter) {
			pc.BlitBox(math32.Vector2{}, sizef, colors.Uniform(colors.White))
			tsty := text.NewStyle()
			fsty := rich.NewStyle()
			fsty.SetFamily(fam)
			tsty.ToDots(&pc.UnitContext)
			y := float32(5)
			for _, ts := range lines {
				tx := rich.NewText(fsty, []rune(ts))
				lns := txtSh.WrapLines(tx, fsty, tsty, &rich.DefaultSettings, sizef)
				pos := math32.Vector2{5, y}
				pc.TextLines(lns, pos)
				y += 20
			}
		})
	}
}

func TestTextMarkup(t *testing.T) {
	size := image.Point{480, 400}
	sizef := math32.FromPoint(size)
	txtSh := shaped.NewShaper()
	RunTest(t, "text/markup", size.X, size.Y, func(pc *Painter) {
		pc.BlitBox(math32.Vector2{}, sizef, colors.Uniform(colors.White))
		tsty := text.NewStyle()
		fsty := rich.NewStyle()
		tsty.FontSize.Dp(60)
		tsty.ToDots(&pc.UnitContext)

		tx, err := htmltext.HTMLToRich([]byte("This is <a>HTML</a> <b>formatted</b> <i>text</i> with <u>underline</u> and <s>strikethrough</s>"), fsty, nil)
		assert.NoError(t, err)
		lns := txtSh.WrapLines(tx, fsty, tsty, &rich.DefaultSettings, sizef)
		lns.SelectRegion(textpos.Range{Start: 5, End: 20})
		// if tsz.X != 100 || tsz.Y != 40 {
		// 	t.Errorf("unexpected text size: %v", tsz)
		// }
		// txt.HasOverflow = true
		pos := math32.Vector2{10, 200}
		pc.Paint.Transform = math32.Rotate2DAround(math32.DegToRad(-45), pos)
		pc.TextLines(lns, pos)
	})
}

func TestTextLines(t *testing.T) {
	size := image.Point{480, 80}
	sizef := math32.FromPoint(size)
	txtSh := shaped.NewShaper()
	RunTest(t, "text/lines", size.X, size.Y, func(pc *Painter) {
		pc.BlitBox(math32.Vector2{}, sizef, colors.Uniform(colors.White))
		tsty := text.NewStyle()
		fsty := rich.NewStyle()
		tsty.FontSize.Dp(16)
		tsty.ToDots(&pc.UnitContext)

		du := *fsty
		du.SetDecoration(rich.DottedUnderline)

		uu := *fsty
		uu.SetDecoration(rich.Underline)

		ol := *fsty
		ol.SetDecoration(rich.Overline)

		fmt.Println("du:", du.Decoration.HasFlag(rich.DottedUnderline), "ol:", du.Decoration.HasFlag(rich.Overline))

		tx := rich.NewText(fsty, []rune("Plain "))
		tx.AddSpan(&du, []rune("Dotted Underline")).AddSpan(fsty, []rune(" and ")).AddSpan(&uu, []rune("Underline"))
		tx.AddSpan(fsty, []rune(" and ")).AddSpan(&ol, []rune("Overline"))

		lns := txtSh.WrapLines(tx, fsty, tsty, &rich.DefaultSettings, sizef)
		pos := math32.Vector2{10, 10}
		// pc.Paint.Transform = math32.Rotate2DAround(math32.DegToRad(-45), pos)
		pc.TextLines(lns, pos)
	})
}
