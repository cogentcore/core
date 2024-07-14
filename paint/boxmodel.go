// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
)

// DrawStandardBox draws the CSS standard box model using the given styling information,
// position, size, and parent actual background. This is used for rendering
// widgets such as buttons, textfields, etc in a GUI.
func (pc *Context) DrawStandardBox(st *styles.Style, pos math32.Vector2, size math32.Vector2, pabg image.Image) {
	if !st.RenderBox {
		return
	}

	pos, size = pc.fixBounds(pos, size)
	tm := st.TotalMargin().Round()
	mpos := pos.Add(tm.Pos())
	msize := size.Sub(tm.Size())
	radius := st.Border.Radius.Dots()

	if st.ActualBackground == nil {
		// we need to do this to prevent
		// elements from rendering over themselves
		// (see https://github.com/cogentcore/core/issues/565)
		st.ActualBackground = pabg
	}

	// note that we always set the fill opacity to 1 because we are already applying
	// the opacity of the background color in ComputeActualBackground above
	pc.FillStyle.Opacity = 1

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
		pc.BlitBox(pos, size, pabg)
	}

	pc.StrokeStyle.Opacity = st.Opacity
	pc.FontStyle.Opacity = st.Opacity

	// first do any shadow
	if st.HasBoxShadow() {
		// CSS effectively goes in reverse order
		for i := len(st.BoxShadow) - 1; i >= 0; i-- {
			shadow := st.BoxShadow[i]
			pc.StrokeStyle.Color = nil
			// note: applying 0.5 here does a reasonable job of matching
			// material design shadows, at their specified alpha levels.
			pc.FillStyle.Color = gradient.ApplyOpacity(shadow.Color, 0.5)
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

			pc.DrawRoundedShadowBlur(shadow.Blur.Dots/2, 1, spos.X, spos.Y, ssz.X, ssz.Y, radius)
		}
	}

	// then draw the box over top of that.
	// we need to draw things twice here because we need to clear
	// the whole area with the background color first so the border
	// doesn't render weirdly
	if styles.SidesAreZero(radius.Sides) {
		pc.FillBox(mpos, msize, st.ActualBackground)
	} else {
		pc.FillStyle.Color = st.ActualBackground
		// no border; fill on
		pc.DrawRoundedRectangle(mpos.X, mpos.Y, msize.X, msize.Y, radius)
		pc.Fill()
	}

	// now that we have drawn background color
	// above, we can draw the border
	mpos.SetSub(st.Border.Width.Dots().Pos().MulScalar(0.5).Ceil())
	msize.SetAdd(st.Border.Width.Dots().Size().MulScalar(0.5).Ceil())
	mpos.SetSub(st.Border.Offset.Dots().Pos().Ceil())
	msize.SetAdd(st.Border.Offset.Dots().Size().Ceil())
	pc.FillStyle.Color = nil
	pc.DrawBorder(mpos.X, mpos.Y, msize.X, msize.Y, st.Border)
}

// fixBounds returns a version of the given position and size such that they
// do not go outside of the parent effective bounds based on their border radius.
// For each corner, if our corner is outside of the inset effective
// border radius corner of our parent, we ensure that our border radius is at
// least as large as that of our parent, thereby ensuring that we do not go
// outside of our parent effective bounds.
func (pc *Context) fixBounds(pos, size math32.Vector2) (math32.Vector2, math32.Vector2) {
	if len(pc.BoundsStack) == 0 {
		return pos, size
	}

	pr := pc.RadiusStack[len(pc.RadiusStack)-1]
	if styles.SidesAreZero(pr.Sides) {
		return pos, size
	}

	pbox := pc.BoundsStack[len(pc.BoundsStack)-1]
	psz := math32.Vector2FromPoint(pbox.Size())
	pr = ClampBorderRadius(pr, psz.X, psz.Y)

	rect := math32.Box2{Min: pos, Max: pos.Add(size)}

	horizFit := func(width, radius float32) (x, y float32) {
		norm := width / radius
		if norm < .5 {
			ang := math32.Acos(norm)
			x = width
			y = radius * math32.Sin(ang)
		} else {
			x = radius * (math32.Sqrt2 / 2)
			y = x
		}
		return
	}
	vertFit := func(height, radius float32) (x, y float32) {
		norm := height / radius
		if norm < .5 {
			ang := math32.Asin(norm)
			x = radius * math32.Cos(ang)
			y = height
		} else {
			x = radius * (math32.Sqrt2 / 2)
			y = x
		}
		return
	}

	// logic is currently based on consistent radius for all corners
	radius := max(pr.Top, pr.Left, pr.Right, pr.Bottom)

	// todo: should this be based on anything?
	extra := float32(2)

	// each of these is how much the element is encroaching into each
	// side of the bounding rectangle, within the radius curve.
	// if the number is negative, then it isn't encroaching at all and can
	// be ignored.
	top := radius - (rect.Min.Y - float32(pbox.Min.Y))
	left := radius - (rect.Min.X - float32(pbox.Min.X))
	right := radius - (float32(pbox.Max.X) - rect.Max.X)
	bottom := radius - (float32(pbox.Max.Y) - rect.Max.Y)

	if top > 0 && left > 0 {
		if left < top {
			x, y := horizFit(left, radius)
			rect.Min.X = max(rect.Min.X, pos.X-(extra+(left-x)))
			rect.Min.Y = max(rect.Min.Y, pos.Y+extra+radius-y)
		} else {
			x, y := vertFit(top, radius)
			rect.Min.X = max(rect.Min.X, pos.X+extra+radius-x)
			rect.Min.Y = max(rect.Min.Y, pos.Y+extra+(top-y))
		}
	}
	if top > 0 && right > 0 {
		if right < top {
			x, y := horizFit(right, radius)
			rect.Max.X = min(rect.Max.X, pos.X+size.X-(extra+(right-x)))
			rect.Min.Y = max(rect.Min.Y, pos.Y+extra+radius-y)
		} else {
			x, y := vertFit(top, radius)
			rect.Max.X = min(rect.Max.X, pos.X+size.X-(extra+radius-x))
			rect.Min.Y = max(rect.Min.Y, pos.Y+extra+(top-y))
		}
	}
	if bottom > 0 && right > 0 {
		if right < bottom {
			x, y := horizFit(right, radius)
			rect.Max.X = min(rect.Max.X, pos.X+size.X-(extra+(right-x)))
			rect.Max.Y = min(rect.Max.Y, pos.Y+size.Y+(extra+radius-y))
		} else {
			x, y := vertFit(bottom, radius)
			rect.Max.X = min(rect.Max.X, pos.X+size.X-(extra+radius-x))
			rect.Max.Y = min(rect.Max.Y, pos.Y+size.Y-(extra+(bottom-y)))
		}
	}
	if bottom > 0 && left > 0 {
		if left < bottom {
			x, y := horizFit(left, radius)
			rect.Min.X = max(rect.Min.X, pos.X-(extra+(left-x)))
			rect.Max.Y = min(rect.Max.Y, pos.Y+size.Y-(extra+radius-y))
		} else {
			x, y := vertFit(bottom, radius)
			rect.Min.X = max(rect.Min.X, pos.X+(extra+radius-x))
			rect.Max.Y = min(rect.Max.Y, pos.Y+size.Y-(extra+(bottom-y)))
		}
	}

	return rect.Min, rect.Max.Sub(rect.Min)
}
