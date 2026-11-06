// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image/color"
	"testing"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
)

// TestHiddenOverContent tests that elements with visibility: hidden or
// opacity: 0 do not paint over the content under them (as the dropdown
// menus of the header of github.com, which are positioned over it).
func TestHiddenOverContent(t *testing.T) {
	for _, hide := range []string{"visibility: hidden", "opacity: 0"} {
		b := core.NewBody()
		ctx := NewContext()
		assert.NoError(t, ReadHTMLString(ctx, b, `<div style="position: relative; width: 200px; height: 100px">`+
			`<div style="background: blue; width: 200px; height: 100px"></div>`+
			`<div style="position: absolute; top: 0; left: 0; width: 200px; height: 100px; background: red; `+hide+`">hidden</div></div>`))
		img := b.RenderImage()
		// the body has a margin of 8px
		c := color.RGBAModel.Convert(img.At(int(50*b.Styles.UnitContext.Dots(1)), int(50*b.Styles.UnitContext.Dots(1)))).(color.RGBA)
		assert.Equal(t, color.RGBA{0, 0, 255, 255}, c, hide)
	}
}
