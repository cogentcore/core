// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cam16

import (
	"image/color"

	"cogentcore.org/core/colors/cam/cie"
	"cogentcore.org/core/math32"
)

// Blend returns a color that is the given percent blend between the first
// and second color; 10 = 10% of the first and 90% of the second, etc;
// blending is done directly on non-premultiplied CAM16-UCS values, and
// a correctly premultiplied color is returned.
func Blend(pct float32, x, y color.Color) color.RGBA {
	pct = math32.Clamp(pct, 0, 100)
	amt := pct / 100

	xsr, xsg, xsb, _ := cie.SRGBUint32ToFloat(x.RGBA())
	ysr, ysg, ysb, _ := cie.SRGBUint32ToFloat(y.RGBA())

	cx := FromSRGB(xsr, xsg, xsb)
	cy := FromSRGB(ysr, ysg, ysb)

	xj, _, xa, xb := cx.UCS()
	yj, _, ya, yb := cy.UCS()

	j := yj + (xj-yj)*amt
	a := ya + (xa-ya)*amt
	b := yb + (xb-yb)*amt

	cam := FromUCS(j, a, b)
	return cam.AsRGBA()
}
