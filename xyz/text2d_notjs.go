// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package xyz

import (
	"image"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
)

func initTextShaper(sc *Scene) {
	sc.TextShaper = shapedgt.NewShaper()
}

// renderTextImage renders the given painter to an image.
func renderTextImage(pc *paint.Painter) *image.RGBA {
	return imagex.AsRGBA(paint.RenderToImage(pc))
}
