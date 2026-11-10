// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"image/color"
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomProperties tests the custom properties of CSS in SVG, used by
// var(--name, fallback) in the style of a node: with the value set for
// :root in a style element, or in the style of a group that it is in, or
// the fallback (as the icons of bicycling.com, of
// style="fill:var(--primary, #000)", which failed to parse as colors).
func TestCustomProperties(t *testing.T) {
	fill := func(src string) color.RGBA {
		sv := NewSVG(math32.Vec2(24, 24))
		require.NoError(t, sv.ReadXML(strings.NewReader(src)))
		sv.Style()
		var c color.RGBA
		sv.Root.WalkDown(func(n tree.Node) bool {
			if p, ok := n.(*Path); ok {
				c = colors.AsRGBA(colors.ToUniform(p.Paint.Fill.Color))
			}
			return tree.Continue
		})
		return c
	}
	const path = `<path d="M0 0h10v10H0z" style="fill:var(--primary, #000)"/>`
	assert.Equal(t, color.RGBA{255, 255, 255, 255}, fill(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24"><style>:root { --primary : #fff; }</style>`+path+`</svg>`), ":root")
	assert.Equal(t, color.RGBA{0, 0, 0, 255}, fill(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24">`+path+`</svg>`), "fallback")
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, fill(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><g style="--primary: var(--red, #f00)">`+path+`</g></svg>`), "group, nested")
}
