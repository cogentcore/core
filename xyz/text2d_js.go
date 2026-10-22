// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package xyz

import (
	"image"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/renderers/htmlcanvas"
	"cogentcore.org/core/text/shaped/shapers/shapedjs"
)

// initTextShaper sets the text shaper of the scene to the browser shaper,
// as the text is also drawn by the browser, in [renderTextImage].
func initTextShaper(sc *Scene) {
	sc.TextShaper = shapedjs.NewShaper()
}

// renderTextImage renders the given painter to an image using an
// offscreen HTML canvas, so that the browser draws the text with
// the same fonts as the rest of the app, which avoids embedding fonts.
func renderTextImage(pc *paint.Painter) *image.RGBA {
	rd := htmlcanvas.NewImage(pc.Size)
	return imagex.AsRGBA(rd.Render(pc.RenderDone()).Image())
}
