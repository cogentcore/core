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
	tm := st.TotalMargin().Round()
	mpos := pos.Add(tm.Pos())
	msize := size.Sub(tm.Size())
	radius := st.Border.Radius.Dots()
	radius = pc.fixRadius(radius, pos, size)

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
			pc.FillStyle.Color = gradient.ApplyOpacityImage(shadow.Color, 0.5)
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
	mpos.SetAdd(st.Border.Width.Dots().Pos().MulScalar(0.5))
	msize.SetSub(st.Border.Width.Dots().Size().MulScalar(0.5))
	mpos.SetSub(st.Border.Offset.Dots().Pos())
	msize.SetAdd(st.Border.Offset.Dots().Size())
	pc.FillStyle.Color = nil
	pc.DrawBorder(mpos.X, mpos.Y, msize.X, msize.Y, st.Border)
}

// fixRadius returns a version of the given border radius that is guaranteed
// to have appropriate values for each corner such that they do not go outside
// of the parent effective bounds.
func (pc *Context) fixRadius(r styles.SideFloats, pos, size math32.Vector2) styles.SideFloats {
	if len(pc.RadiusStack) == 0 {
		return r
	}
	rtop := pos.ToPoint()
	rright := pos.AddDim(math32.X, size.X).ToPoint()
	rbottom := pos.Add(size).ToPoint()
	rleft := pos.AddDim(math32.Y, size.Y).ToPoint()

	// For each parent and corner, if our corner is outside of the inset effective
	// border radius corner of the parent, we ensure that our border radius is at
	// least as large as that of the parent, thereby ensuring that we do not go
	// outside of the parent effective bounds.
	for i, pbox := range pc.ContentBoundsStack {
		pr := pc.RadiusStack[i]

		ptop := pbox.Min.Add(image.Pt(int(pr.Top), int(pr.Top)))
		if rtop.X < ptop.X && rtop.Y < ptop.Y {
			r.Top = max(r.Top, pr.Top)
		}

		pright := pbox.Min.Add(image.Pt(pbox.Size().X-int(pr.Right), int(pr.Right)))
		if rright.X > pright.X && rright.Y < pright.Y {
			r.Right = max(r.Right, pr.Right)
		}

		pbottom := pbox.Min.Add(image.Pt(pbox.Size().X-int(pr.Bottom), pbox.Size().Y-int(pr.Bottom)))
		if rbottom.X > pbottom.X && rbottom.Y > pbottom.Y {
			r.Bottom = max(r.Bottom, pr.Bottom)
		}

		pleft := pbox.Min.Add(image.Pt(int(pr.Left), pbox.Size().Y-int(pr.Left)))
		if rleft.X < pleft.X && rleft.Y > pleft.Y {
			r.Left = max(r.Left, pr.Left)
		}
	}
	return r
}
