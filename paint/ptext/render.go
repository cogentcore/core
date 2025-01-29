// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ptext

import (
	"image"
	"unicode"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/f64"
)

// Render actually does text rendering into given image, using all data
// stored previously during PreRender, and using given renderer to draw
// the text path decorations etc.
func (tr *Text) Render(img *image.RGBA, rd render.Renderer) {
	// pr := profile.Start("RenderText")
	// defer pr.End()

	ctx := &tr.Context
	var ppaint styles.Paint
	ppaint.CopyStyleFrom(&ctx.Style)
	cb := ctx.Bounds.Rect.ToRect()
	pos := tr.RenderPos

	// todo:
	// pc.PushTransform(math32.Identity2()) // needed for SVG
	// defer pc.PopTransform()
	ctx.Transform = math32.Identity2()

	TextFontRenderMu.Lock()
	defer TextFontRenderMu.Unlock()

	elipses := '…'
	hadOverflow := false
	rendOverflow := false
	overBoxSet := false
	var overStart math32.Vector2
	var overBox math32.Box2
	var overFace font.Face
	var overColor image.Image

	for si := range tr.Spans {
		sr := &tr.Spans[si]
		if sr.IsValid() != nil {
			continue
		}

		curFace := sr.Render[0].Face
		curColor := sr.Render[0].Color
		if g, ok := curColor.(gradient.Gradient); ok {
			_ = g
			// todo: no last render bbox:
			// g.Update(pc.FontStyle.Opacity, math32.B2FromRect(pc.LastRenderBBox), pc.Transform)
		} else {
			curColor = gradient.ApplyOpacity(curColor, ctx.Style.FontStyle.Opacity)
		}
		tpos := pos.Add(sr.RelPos)

		if !overBoxSet {
			overWd, _ := curFace.GlyphAdvance(elipses)
			overWd32 := math32.FromFixed(overWd)
			overEnd := math32.FromPoint(cb.Max)
			overStart = overEnd.Sub(math32.Vec2(overWd32, 0.1*tr.FontHeight))
			overBox = math32.Box2{Min: math32.Vec2(overStart.X, overEnd.Y-tr.FontHeight), Max: overEnd}
			overFace = curFace
			overColor = curColor
			overBoxSet = true
		}

		d := &font.Drawer{
			Dst:  img,
			Src:  curColor,
			Face: curFace,
		}

		rd.Render(sr.BgPaths)
		rd.Render(sr.DecoPaths)

		for i, r := range sr.Text {
			rr := &(sr.Render[i])
			if rr.Color != nil {
				curColor := rr.Color
				curColor = gradient.ApplyOpacity(curColor, ctx.Style.FontStyle.Opacity)
				d.Src = curColor
			}
			curFace = rr.CurFace(curFace)
			if !unicode.IsPrint(r) {
				continue
			}
			dsc32 := math32.FromFixed(curFace.Metrics().Descent)
			rp := tpos.Add(rr.RelPos)
			scx := float32(1)
			if rr.ScaleX != 0 {
				scx = rr.ScaleX
			}
			tx := math32.Scale2D(scx, 1).Rotate(rr.RotRad)
			ll := rp.Add(tx.MulVector2AsVector(math32.Vec2(0, dsc32)))
			ur := ll.Add(tx.MulVector2AsVector(math32.Vec2(rr.Size.X, -rr.Size.Y)))

			if int(math32.Ceil(ur.X)) < cb.Min.X || int(math32.Ceil(ll.Y)) < cb.Min.Y {
				continue
			}

			doingOverflow := false
			if tr.HasOverflow {
				cmid := ll.Add(math32.Vec2(0.5*rr.Size.X, -0.5*rr.Size.Y))
				if overBox.ContainsPoint(cmid) {
					doingOverflow = true
					r = elipses
				}
			}

			if int(math32.Floor(ll.X)) > cb.Max.X+1 || int(math32.Floor(ur.Y)) > cb.Max.Y+1 {
				hadOverflow = true
				if !doingOverflow {
					continue
				}
			}

			if rendOverflow { // once you've rendered, no more rendering
				continue
			}

			d.Face = curFace
			d.Dot = rp.ToFixed()
			dr, mask, maskp, _, ok := d.Face.Glyph(d.Dot, r)
			if !ok {
				// fmt.Printf("not ok rendering rune: %v\n", string(r))
				continue
			}
			if rr.RotRad == 0 && (rr.ScaleX == 0 || rr.ScaleX == 1) {
				idr := dr.Intersect(cb)
				soff := image.Point{}
				if dr.Min.X < cb.Min.X {
					soff.X = cb.Min.X - dr.Min.X
					maskp.X += cb.Min.X - dr.Min.X
				}
				if dr.Min.Y < cb.Min.Y {
					soff.Y = cb.Min.Y - dr.Min.Y
					maskp.Y += cb.Min.Y - dr.Min.Y
				}
				draw.DrawMask(d.Dst, idr, d.Src, soff, mask, maskp, draw.Over)
			} else {
				srect := dr.Sub(dr.Min)
				dbase := math32.Vec2(rp.X-float32(dr.Min.X), rp.Y-float32(dr.Min.Y))

				transformer := draw.BiLinear
				fx, fy := float32(dr.Min.X), float32(dr.Min.Y)
				m := math32.Translate2D(fx+dbase.X, fy+dbase.Y).Scale(scx, 1).Rotate(rr.RotRad).Translate(-dbase.X, -dbase.Y)
				s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
				transformer.Transform(d.Dst, s2d, d.Src, srect, draw.Over, &draw.Options{
					SrcMask:  mask,
					SrcMaskP: maskp,
				})
			}
			if doingOverflow {
				rendOverflow = true
			}
		}
		rd.Render(sr.StrikePaths)
	}
	tr.HasOverflow = hadOverflow

	if hadOverflow && !rendOverflow && overBoxSet {
		d := &font.Drawer{
			Dst:  img,
			Src:  overColor,
			Face: overFace,
			Dot:  overStart.ToFixed(),
		}
		dr, mask, maskp, _, _ := d.Face.Glyph(d.Dot, elipses)
		idr := dr.Intersect(cb)
		soff := image.Point{}
		draw.DrawMask(d.Dst, idr, d.Src, soff, mask, maskp, draw.Over)
	}
}
