// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/grows/images"
	"goki.dev/grr"
	"goki.dev/mat32/v2"
)

// DrawBox calls DrawBorder with position, size and border parameters
// as a convenience method for DrawStdBox
func (pc *Paint) DrawBox(rs *State, pos mat32.Vec2, sz mat32.Vec2, bs styles.Border) {
	pc.DrawBorder(rs, pos.X, pos.Y, sz.X, sz.Y, bs)
}

// DrawStdBox draws the CSS "standard box" model using given style.
// This is used for rendering widgets such as buttons, textfields, etc in a GUI.
// The surround arguments are the background color and state layer of the surrounding
// context of this box, typically obtained through [goki.dev/gi/v2/gi.WidgetBase.ParentBackgroundColor]
// in a GUI context.
func (pc *Paint) DrawStdBox(rs *State, st *styles.Style, pos mat32.Vec2, sz mat32.Vec2, surroundBgColor *colors.Full, surroundStateLayer float32) {
	mpos := pos.Add(st.TotalMargin().Pos())
	msz := sz.Sub(st.TotalMargin().Size())
	rad := st.Border.Radius.Dots()
	_ = rad

	// the background color we actually use
	bg := st.BackgroundColor
	if bg.IsNil() {
		// we need to do this to prevent
		// elements from rendering over themselves
		// (see https://github.com/goki/gi/issues/565)
		bg = *surroundBgColor
	}
	// we need to apply the state layer after getting the
	// surrounding background color
	bg = st.StateBackgroundColor(bg)

	// we only fill the surrounding background color if we are told to
	if st.FillMargin {
		// we apply the surrounding state layer to the surrounding background color
		psl := st.StateLayer
		st.StateLayer = surroundStateLayer
		sbg := st.StateBackgroundColor(*surroundBgColor)
		st.StateLayer = psl

		// We need to fill the whole box where the
		// box shadows / element can go to prevent growing
		// box shadows and borders. We couldn't just
		// do this when there are box shadows, as they
		// may be removed and then need to be covered up.
		// This also fixes https://github.com/goki/gi/issues/579.
		// This isn't an ideal solution because of performance,
		// so TODO: maybe come up with a better solution for this.
		// We need to use raw LayState data because we need to clear
		// any box shadow that may have gone in margin.
		pc.FillBox(rs, pos, sz, &sbg)
	}

	// first do any shadow
	if st.HasBoxShadow() {
		// CSS effectively goes in reverse order
		for i := len(st.BoxShadow) - 1; i >= 0; i-- {
			shadow := st.BoxShadow[i]
			pc.StrokeStyle.SetColor(nil)
			prevOpacity := pc.FillStyle.Opacity
			// note: factor of 0.5 here does a reasonable job of matching
			// material design shadows, at their specified alpha levels.
			pc.FillStyle.Opacity = (float32(shadow.Color.A) / 255) * .5
			pc.FillStyle.SetColor(colors.SetA(shadow.Color, 255))
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

			pc.DrawRoundedShadowBlur(rs, shadow.Blur.Dots/2, 1, spos.X, spos.Y, ssz.X, ssz.Y, st.Border.Radius.Dots())
			pc.FillStyle.Opacity = prevOpacity
		}
	}

	// then draw the box over top of that.
	// need to set clipping to box first.. (?)
	// we need to draw things twice here because we need to clear
	// the whole area with the background color first so the border
	// doesn't render weirdly
	if st.BackgroundImage != nil {
		img, _, err := images.Read(st.BackgroundImage)
		if grr.Log(err) == nil {
			// TODO(kai/girl): image scaling
			pc.DrawImage(rs, img, mpos.X, mpos.Y)
		}
	} else {
		if rad.IsZero() {
			pc.FillBox(rs, mpos, msz, &bg)
		} else {
			pc.FillStyle.SetFullColor(&bg)
			// no border -- fill only
			pc.DrawRoundedRectangle(rs, mpos.X, mpos.Y, msz.X, msz.Y, rad)
			pc.Fill(rs)
		}
	}

	// pc.StrokeStyle.SetColor(&st.Border.Color)
	// pc.StrokeStyle.Width = st.Border.Width
	// pc.FillStyle.SetFullColor(&st.BackgroundColor)
	mpos.SetAdd(st.Border.Width.Dots().Pos().MulScalar(0.5))
	msz.SetSub(st.Border.Width.Dots().Size().MulScalar(0.5))
	pc.FillStyle.SetColor(nil)
	// now that we have drawn background color
	// above, we can draw the border
	pc.DrawBox(rs, mpos, msz, st.Border)
}
