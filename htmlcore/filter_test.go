// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image/color"
	"testing"

	"cogentcore.org/core/htmlcore/css"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterColors(t *testing.T) {
	grey := color.NRGBA{95, 99, 104, 255}
	white := css.ParseFilter("brightness(0) saturate(1) invert(1)").Color(grey)
	assert.Equal(t, color.NRGBA{255, 255, 255, 255}, white, "the wedges of pkg.go.dev")
	assert.Equal(t, color.NRGBA{160, 156, 151, 255}, css.ParseFilter("invert(100%)").Color(grey))
	half := css.ParseFilter("opacity(50%)").Color(grey)
	assert.Equal(t, uint8(128), half.A)
	g := css.ParseFilter("grayscale(1)").Color(color.NRGBA{255, 0, 0, 255})
	assert.Equal(t, g.R, g.G, "gray")
	assert.Nil(t, css.ParseFilter("none"))
	assert.Nil(t, css.ParseFilter("blur(2px)"), "not a color function")
}

// TestFilterSVGImage tests that an img of an svg with a filter shows the
// filtered colors (as the grey wedges of pkg.go.dev, which are white).
func TestFilterSVGImage(t *testing.T) {
	wedge := `data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"%3E%3Crect width="24" height="24" fill="%235f6368"/%3E%3C/svg%3E`
	b := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), b, `<div style="background: black; padding: 10px"><img src='`+wedge+`' width="24" height="24" style="filter: brightness(0) saturate(1) invert(1)"></div>`))
	var im *core.Image
	b.WalkDown(func(n tree.Node) bool {
		if i, ok := n.(*core.Image); ok {
			im = i
		}
		return tree.Continue
	})
	require.NotNil(t, im, "an image of the filtered svg")
	img := core.RenderImage(b)
	bb := im.Geom.TotalBBox
	c := bb.Min.Add(bb.Size().Div(2))
	assert.Equal(t, color.RGBA{255, 255, 255, 255}, color.RGBAModel.Convert(img.At(c.X, c.Y)), "white")
	assert.InDelta(t, 24*b.Styles.UnitContext.Dots(units.UnitDp), float32(bb.Dx()), 2, "its size")
}
