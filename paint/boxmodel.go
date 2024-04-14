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

// DrawStandardBox draws the CSS "standard box" model using the given styling information,
// position, size, and parent actual background. This is used for rendering
// widgets such as buttons, textfields, etc in a GUI.
func (pc *Context) DrawStandardBox(st *styles.Style, pos math32.Vector2, sz math32.Vector2, pabg image.Image) {
	tm := st.TotalMargin().Round()
	mpos := pos.Add(tm.Pos())
	msz := sz.Sub(tm.Size())
	rad := st.Border.Radius.Dots()

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
		pc.BlitBox(pos, sz, pabg)
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
			pc.FillStyle.Color = gradient.ApplyOpacityImage(shadow.Color, 0.5)
			spos := shadow.BasePos(mpos)
			ssz := shadow.BaseSize(msz)

			// note: we are using EdgeBlurFactors with radiusFactor = 1
			// (sigma == radius), so we divide Blur / 2 relative to the
			// CSS standard of sigma = blur / 2 (i.e., our sigma = blur,
			// so we divide Blur / 2 to achieve the same effect).
			// This works fine for low-opacity blur factors (the edges are
			// so transparent that you can't really see beyond 1 sigma,
			// if you used radiusFactor = 2).
			// If a higher-contrast shadow is used, it would look better
			// with radiusFactor = 2, and you'd have to remove this /2 factor.

			pc.DrawRoundedShadowBlur(shadow.Blur.Dots/2, 1, spos.X, spos.Y, ssz.X, ssz.Y, st.Border.Radius.Dots())
		}
	}

	// then draw the box over top of that.
	// we need to draw things twice here because we need to clear
	// the whole area with the background color first so the border
	// doesn't render weirdly
	if styles.SidesAreZero(rad.Sides) {
		pc.FillBox(mpos, msz, st.ActualBackground)
	} else {
		pc.FillStyle.Color = st.ActualBackground
		// no border; fill on
		pc.DrawRoundedRectangle(mpos.X, mpos.Y, msz.X, msz.Y, rad)
		pc.Fill()
	}

	// now that we have drawn background color
	// above, we can draw the border
	mpos.SetAdd(st.Border.Width.Dots().Pos().MulScalar(0.5))
	msz.SetSub(st.Border.Width.Dots().Size().MulScalar(0.5))
	mpos.SetSub(st.Border.Offset.Dots().Pos())
	msz.SetAdd(st.Border.Offset.Dots().Size())
	pc.FillStyle.Color = nil
	pc.DrawBorder(mpos.X, mpos.Y, msz.X, msz.Y, st.Border)
}
