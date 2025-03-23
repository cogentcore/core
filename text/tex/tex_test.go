// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex

import (
	"image/color"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	. "cogentcore.org/core/paint"
	_ "cogentcore.org/core/paint/renderers"
	"github.com/stretchr/testify/assert"
)

// RunTest makes a rendering state, paint, and image with the given size, calls the given
// function, and then asserts the image using [imagex.Assert] with the given name.
func RunTest(t *testing.T, nm string, width int, height int, f func(pc *Painter)) {
	pc := NewPainter(width, height)
	pc.FillBox(math32.Vector2{}, math32.Vec2(float32(width), float32(height)), colors.Uniform(colors.White))
	f(pc)
	pc.RenderToImage()
	imagex.Assert(t, pc.RenderImage(), nm)
}

func TestTex(t *testing.T) {
	RunTest(t, "basic", 1000, 1000, func(pc *Painter) {
		pc.Fill.Color = colors.Uniform(color.Black)
		pp, err := ParseLaTeX("a = x")
		assert.NoError(t, err)
		pc.State.Path = *pp
		pc.PathDone()
	})
}
