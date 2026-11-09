// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
)

// StandardBox draws the CSS standard box model using the given styling information,
// position, size, and parent actual background. This is used for rendering
// widgets such as buttons, text fields, etc in a GUI.
func (pc *Painter) StandardBox(st *styles.Style, pos math32.Vector2, size math32.Vector2, pabg image.Image) {
	if !st.RenderBox || size == (math32.Vector2{}) {
		return
	}

	encroach, pr := pc.boundsEncroachParent(pos, size)
	tm := st.TotalMargin().Round()
	mpos := pos.Add(tm.Pos())
	msize := size.Sub(tm.Size())
	if msize == (math32.Vector2{}) && sides.AreZero(st.Border.Width.Dots().Sides) {
		return // a box of no size is still rendered with its borders (as a CSS triangle)
	}
	radius := st.Border.Radius.Dots()
	if encroach { // if we encroach, we must limit ourselves to the parent radius
		radius = radius.Max(pr)
	}

	if st.ActualBackground == nil {
		// we need to do this to prevent
		// elements from rendering over themselves
		// (see https://github.com/cogentcore/core/issues/565)
		st.ActualBackground = pabg
	}

	// note that we always set the fill opacity to 1 because we are already applying
	// the opacity of the background color in ComputeActualBackground above
	pc.Fill.Opacity = 1

	if st.FillMargin {
		// We need to fill the whole box where the
		// box shadows / element can go to prevent growing
		// box shadows and borders. We couldn't just
		// do this when there are box shadows, as they
		// may be removed and then need to be covered up.
		// This also fixes https://github.com/cogentcore/core/issues/579.
		// This isn't an ideal solution because of performance,
		// so TODO: maybe come up with a better solution for this.
		// We need to use raw geom data because we need to clear
		// any box shadow that may have gone in margin.
		if encroach { // if we encroach, we must limit ourselves to the parent radius
			pc.Fill.Color = pabg
			pc.RoundedRectangleSides(pos.X, pos.Y, size.X, size.Y, radius)
			pc.Draw()
		} else {
			pc.BlitBox(pos, size, pabg)
		}
	}

	pc.Stroke.Opacity = st.Opacity
	// pc.Font.Opacity = st.Opacity // todo:

	// first do any shadow
	if st.HasBoxShadow() {
		// CSS effectively goes in reverse order
		for i := len(st.BoxShadow) - 1; i >= 0; i-- {
			shadow := st.BoxShadow[i]
			pc.Stroke.Color = nil
			// note: applying 0.5 here does a reasonable job of matching
			// material design shadows, at their specified alpha levels.
			if st.BoxShadowOutside { // CSS shadows have their color
				pc.Fill.Color = shadow.Color
			} else {
				pc.Fill.Color = gradient.ApplyOpacity(shadow.Color, 0.5)
			}
			spos := shadow.BasePos(mpos)
			ssz := shadow.BaseSize(msize)

			// note: we are using EdgeBlurFactors with radiusFactor = 1
			// (sigma == radius), so we divide Blur / 2 relative to the
			// CSS standard of sigma = blur / 2 (i.e., our sigma = blur,
			// so we divide Blur / 2 to achieve the same effect).
			// This works fine for low-opacity blur factors (the edges are
			// so transparent that you can't really see beyond 1 sigma,
			// if you used radiusFactor = 2).
			// If a higher-contrast shadow is used, it would look better
			// with radiusFactor = 2, and you'd have to remove this /2 factor.

			pc.RoundedShadowBlur(shadow.Blur.Dots/2, 1, spos.X, spos.Y, ssz.X, ssz.Y, radius)
		}
	}

	// then draw the box over top of that.
	// we need to draw things twice here because we need to clear
	// the whole area with the background color first so the border
	// doesn't render weirdly
	if sides.AreZero(radius.Sides) {
		pc.FillBox(mpos, msize, st.ActualBackground)
	} else {
		pc.Fill.Color = st.ActualBackground
		// no border; fill on
		pc.RoundedRectangleSides(mpos.X, mpos.Y, msize.X, msize.Y, radius)
		pc.Draw()
	}

	// now that we have drawn background color
	// above, we can draw the border
	if pc.borderSides(mpos, msize, radius, &st.Border) {
		return
	}
	mpos.SetSub(st.Border.Width.Dots().Pos().MulScalar(0.5))
	msize.SetAdd(st.Border.Width.Dots().Size().MulScalar(0.5))
	mpos.SetSub(st.Border.Offset.Dots().Pos())
	msize.SetAdd(st.Border.Offset.Dots().Size())
	pc.Fill.Color = nil
	pc.Border(mpos.X, mpos.Y, msize.X, msize.Y, st.Border)
}

// boundsEncroachParent returns whether the current box encroaches on the
// parent bounds, taking into account the parent radius, which is also returned.
func (pc *Painter) boundsEncroachParent(pos, size math32.Vector2) (bool, sides.Floats) {
	if len(pc.Stack) <= 1 {
		return false, sides.Floats{}
	}

	ctx := pc.Stack[len(pc.Stack)-2]
	pr := ctx.Bounds.Radius
	if sides.AreZero(pr.Sides) {
		return false, pr
	}

	pbox := ctx.Bounds.Rect.ToRect()
	psz := ctx.Bounds.Rect.Size()
	pr = ClampBorderRadius(pr, psz.X, psz.Y)

	rect := math32.Box2{Min: pos, Max: pos.Add(size)}

	// logic is currently based on consistent radius for all corners
	radius := max(pr.Top, pr.Left, pr.Right, pr.Bottom)

	// each of these is how much the element is encroaching into each
	// side of the bounding rectangle, within the radius curve.
	// if the number is negative, then it isn't encroaching at all and can
	// be ignored.
	top := radius - (rect.Min.Y - float32(pbox.Min.Y))
	left := radius - (rect.Min.X - float32(pbox.Min.X))
	right := radius - (float32(pbox.Max.X) - rect.Max.X)
	bottom := radius - (float32(pbox.Max.Y) - rect.Max.Y)

	return top > 0 || left > 0 || right > 0 || bottom > 0, pr
}

// borderSides draws the sides of the given border around the given inner
// (padding) box, if they differ in color or width and the box has square
// corners and solid borders, and returns whether it did: as in CSS, each
// side is filled between the outer and inner edges of the border, with
// mitered corners, which makes triangles of the borders of a box of no
// size with transparent sides (as the arrows of selects in CSS).
func (pc *Painter) borderSides(pos, size math32.Vector2, radius sides.Floats, bs *styles.Border) bool {
	w := bs.Width.Dots()
	if !sides.AreZero(radius.Sides) || !sides.AreZero(bs.Offset.Dots().Sides) {
		return false
	}
	if sides.AreSame(bs.Color) && sides.AreSame(w.Sides) {
		return false
	}
	for _, st := range []styles.BorderStyles{bs.Style.Top, bs.Style.Right, bs.Style.Bottom, bs.Style.Left} {
		if st != styles.BorderSolid && st != styles.BorderNone {
			return false
		}
	}
	ix0, iy0 := pos.X, pos.Y
	ix1, iy1 := pos.X+size.X, pos.Y+size.Y
	ox0, oy0 := ix0-w.Left, iy0-w.Top
	ox1, oy1 := ix1+w.Right, iy1+w.Bottom
	side := func(width float32, st styles.BorderStyles, clr image.Image, pts ...float32) {
		if width <= 0 || st == styles.BorderNone || clr == nil {
			return
		}
		pc.MoveTo(pts[0], pts[1])
		for i := 2; i < len(pts); i += 2 {
			pc.LineTo(pts[i], pts[i+1])
		}
		pc.Close()
		pc.Stroke.Color = nil
		pc.Fill.Color = clr
		pc.Draw()
	}
	side(w.Top, bs.Style.Top, bs.Color.Top, ox0, oy0, ox1, oy0, ix1, iy0, ix0, iy0)
	side(w.Right, bs.Style.Right, bs.Color.Right, ox1, oy0, ox1, oy1, ix1, iy1, ix1, iy0)
	side(w.Bottom, bs.Style.Bottom, bs.Color.Bottom, ox1, oy1, ox0, oy1, ix0, iy1, ix1, iy1)
	side(w.Left, bs.Style.Left, bs.Color.Left, ox0, oy1, ox0, oy0, ix0, iy0, ix0, iy1)
	pc.Fill.Color = nil
	return true
}
