// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint_test

import (
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

func TestText(t *testing.T) {
	size := image.Point{480, 400}
	sizef := math32.FromPoint(size)
	txtSh := shaped.NewShaper()
	RunTest(t, "text", size.X, size.Y, func(pc *Painter) {
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
