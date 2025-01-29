// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ptext

import (
	"unicode"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
)

// PreRender performs pre-rendering steps based on a fully-configured
// Text layout. It generates the Path elements for rendering, recording given
// absolute position offset (specifying position of text baseline).
// Any applicable transforms (aside from the char-specific rotation in Render)
// must be applied in advance in computing the relative positions of the
// runes, and the overall font size, etc.
func (tr *Text) PreRender(ctx *render.Context, pos math32.Vector2) {
	// ctx.Transform = math32.Identity2()
	tr.Context = *ctx
	tr.RenderPos = pos

	for si := range tr.Spans {
		sr := &tr.Spans[si]
		if sr.IsValid() != nil {
			continue
		}
		tpos := pos.Add(sr.RelPos)
		sr.DecoPaths = sr.DecoPaths[:0]
		sr.BgPaths = sr.BgPaths[:0]
		sr.StrikePaths = sr.StrikePaths[:0]
		if sr.HasDeco.HasFlag(styles.DecoBackgroundColor) {
			sr.RenderBg(ctx, tpos)
		}
		if sr.HasDeco.HasFlag(styles.Underline) || sr.HasDeco.HasFlag(styles.DecoDottedUnderline) {
			sr.RenderUnderline(ctx, tpos)
		}
		if sr.HasDeco.HasFlag(styles.Overline) {
			sr.RenderLine(ctx, tpos, styles.Overline, 1.1)
		}
		if sr.HasDeco.HasFlag(styles.LineThrough) {
			sr.RenderLine(ctx, tpos, styles.LineThrough, 0.25)
		}
	}
}

// RenderBg adds renders for the background behind chars.
func (sr *Span) RenderBg(ctx *render.Context, tpos math32.Vector2) {
	curFace := sr.Render[0].Face
	didLast := false
	cb := ctx.Bounds.Rect.ToRect()
	p := ppath.Path{}
	nctx := *ctx
	for i := range sr.Text {
		rr := &(sr.Render[i])
		if rr.Background == nil {
			if didLast {
				sr.BgPaths.Add(render.NewPath(p, &nctx))
				p = ppath.Path{}
			}
			didLast = false
			continue
		}
		curFace = rr.CurFace(curFace)
		dsc32 := math32.FromFixed(curFace.Metrics().Descent)
		rp := tpos.Add(rr.RelPos)
		scx := float32(1)
		if rr.ScaleX != 0 {
			scx = rr.ScaleX
		}
		tx := math32.Scale2D(scx, 1).Rotate(rr.RotRad)
		ll := rp.Add(tx.MulVector2AsVector(math32.Vec2(0, dsc32)))
		ur := ll.Add(tx.MulVector2AsVector(math32.Vec2(rr.Size.X, -rr.Size.Y)))
		if int(math32.Floor(ll.X)) > cb.Max.X || int(math32.Floor(ur.Y)) > cb.Max.Y ||
			int(math32.Ceil(ur.X)) < cb.Min.X || int(math32.Ceil(ll.Y)) < cb.Min.Y {
			if didLast {
				sr.BgPaths.Add(render.NewPath(p, &nctx))
				p = ppath.Path{}
			}
			didLast = false
			continue
		}

		szt := math32.Vec2(rr.Size.X, -rr.Size.Y)
		sp := rp.Add(tx.MulVector2AsVector(math32.Vec2(0, dsc32)))
		ul := sp.Add(tx.MulVector2AsVector(math32.Vec2(0, szt.Y)))
		lr := sp.Add(tx.MulVector2AsVector(math32.Vec2(szt.X, 0)))
		nctx.Style.Fill.Color = rr.Background
		p = p.Append(ppath.Polygon(sp, ul, ur, lr))
		didLast = true
	}
	if didLast {
		sr.BgPaths.Add(render.NewPath(p, &nctx))
	}
}

// RenderUnderline renders the underline for span -- ensures continuity to do it all at once
func (sr *Span) RenderUnderline(ctx *render.Context, tpos math32.Vector2) {
	curFace := sr.Render[0].Face
	curColor := sr.Render[0].Color
	didLast := false
	cb := ctx.Bounds.Rect.ToRect()
	nctx := *ctx
	p := ppath.Path{}

	for i, r := range sr.Text {
		if !unicode.IsPrint(r) {
			continue
		}
		rr := &(sr.Render[i])
		if !(rr.Deco.HasFlag(styles.Underline) || rr.Deco.HasFlag(styles.DecoDottedUnderline)) {
			if didLast {
				sr.DecoPaths.Add(render.NewPath(p, &nctx))
				p = ppath.Path{}
				didLast = false
			}
			continue
		}
		curFace = rr.CurFace(curFace)
		if rr.Color != nil {
			curColor = rr.Color
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
		if int(math32.Floor(ll.X)) > cb.Max.X || int(math32.Floor(ur.Y)) > cb.Max.Y ||
			int(math32.Ceil(ur.X)) < cb.Min.X || int(math32.Ceil(ll.Y)) < cb.Min.Y {
			if didLast {
				sr.DecoPaths.Add(render.NewPath(p, &nctx))
				p = ppath.Path{}
				didLast = false
			}
			continue
		}
		dw := .05 * rr.Size.Y
		if !didLast {
			nctx.Style.Stroke.Width.Dots = dw
			nctx.Style.Stroke.Color = curColor
			if rr.Deco.HasFlag(styles.DecoDottedUnderline) {
				nctx.Style.Stroke.Dashes = []float32{2, 2}
			}
		}
		sp := rp.Add(tx.MulVector2AsVector(math32.Vec2(0, 2*dw)))
		ep := rp.Add(tx.MulVector2AsVector(math32.Vec2(rr.Size.X, 2*dw)))

		if didLast {
			p.LineTo(sp.X, sp.Y)
		} else {
			p.MoveTo(sp.X, sp.Y)
		}
		p.LineTo(ep.X, ep.Y)
		didLast = true
	}
	if didLast {
		sr.DecoPaths.Add(render.NewPath(p, &nctx))
		p = ppath.Path{}
	}
}

// RenderLine renders overline or line-through -- anything that is a function of ascent
func (sr *Span) RenderLine(ctx *render.Context, tpos math32.Vector2, deco styles.TextDecorations, ascPct float32) {
	curFace := sr.Render[0].Face
	curColor := sr.Render[0].Color
	var rend render.Render
	didLast := false
	cb := ctx.Bounds.Rect.ToRect()
	nctx := *ctx
	p := ppath.Path{}

	for i, r := range sr.Text {
		if !unicode.IsPrint(r) {
			continue
		}
		rr := &(sr.Render[i])
		if !rr.Deco.HasFlag(deco) {
			if didLast {
				rend.Add(render.NewPath(p, &nctx))
				p = ppath.Path{}
			}
			didLast = false
			continue
		}
		curFace = rr.CurFace(curFace)
		dsc32 := math32.FromFixed(curFace.Metrics().Descent)
		asc32 := math32.FromFixed(curFace.Metrics().Ascent)
		rp := tpos.Add(rr.RelPos)
		scx := float32(1)
		if rr.ScaleX != 0 {
			scx = rr.ScaleX
		}
		tx := math32.Scale2D(scx, 1).Rotate(rr.RotRad)
		ll := rp.Add(tx.MulVector2AsVector(math32.Vec2(0, dsc32)))
		ur := ll.Add(tx.MulVector2AsVector(math32.Vec2(rr.Size.X, -rr.Size.Y)))
		if int(math32.Floor(ll.X)) > cb.Max.X || int(math32.Floor(ur.Y)) > cb.Max.Y ||
			int(math32.Ceil(ur.X)) < cb.Min.X || int(math32.Ceil(ll.Y)) < cb.Min.Y {
			if didLast {
				rend.Add(render.NewPath(p, &nctx))
				p = ppath.Path{}
			}
			continue
		}
		if rr.Color != nil {
			curColor = rr.Color
		}
		dw := 0.05 * rr.Size.Y
		if !didLast {
			nctx.Style.Stroke.Width.Dots = dw
			nctx.Style.Stroke.Color = curColor
		}
		yo := ascPct * asc32
		sp := rp.Add(tx.MulVector2AsVector(math32.Vec2(0, -yo)))
		ep := rp.Add(tx.MulVector2AsVector(math32.Vec2(rr.Size.X, -yo)))

		if didLast {
			p.LineTo(sp.X, sp.Y)
		} else {
			p.MoveTo(sp.X, sp.Y)
		}
		p.LineTo(ep.X, ep.Y)
		didLast = true
	}
	if didLast {
		rend.Add(render.NewPath(p, &nctx))
	}
	if deco.HasFlag(styles.LineThrough) {
		sr.StrikePaths = rend
	} else {
		sr.DecoPaths = append(sr.DecoPaths, rend...)
	}
}
