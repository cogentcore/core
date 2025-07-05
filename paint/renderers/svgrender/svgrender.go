// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svgrender

import (
	"bytes"
	"image"
	"maps"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
)

// Renderer is the SVG renderer.
type Renderer struct {
	size math32.Vector2

	SVG *svg.SVG

	// gpStack is a stack of groups used while building the svg
	gpStack stack.Stack[*svg.Group]
}

func New(size math32.Vector2) render.Renderer {
	rs := &Renderer{}
	rs.SetSize(units.UnitDot, size)
	return rs
}

func (rs *Renderer) Image() image.Image {
	if rs.SVG == nil {
		return nil
	}
	pc := rs.SVG.Render(nil)
	ir := paint.NewImageRenderer(rs.size)
	ir.Render(pc.RenderDone())
	return ir.Image()
}

func (rs *Renderer) Source() []byte {
	if rs.SVG == nil {
		return nil
	}
	var b bytes.Buffer
	rs.SVG.WriteXML(&b, true)
	return b.Bytes()
}

func (rs *Renderer) Size() (units.Units, math32.Vector2) {
	return units.UnitDot, rs.size
}

func (rs *Renderer) SetSize(un units.Units, size math32.Vector2) {
	if rs.size == size {
		return
	}
	rs.size = size
}

// Render is the main rendering function.
func (rs *Renderer) Render(r render.Render) render.Renderer {
	rs.SVG = svg.NewSVG(rs.size)
	rs.gpStack = nil
	bg := svg.NewGroup(rs.SVG.Root)
	rs.gpStack.Push(bg)
	for _, ri := range r {
		switch x := ri.(type) {
		case *render.Path:
			rs.RenderPath(x)
		case *pimage.Params:
			rs.RenderImage(x)
		case *render.Text:
			rs.RenderText(x)
		case *render.ContextPush:
			rs.PushContext(x)
		case *render.ContextPop:
			rs.PopContext(x)
		}
	}
	// pc := paint.NewPainter(rs.size)
	// rs.SVG.Render(pc)
	// rs.rend = pc.RenderDone()
	return rs
}

func (rs *Renderer) PushGroup() *svg.Group {
	cg := rs.gpStack.Peek()
	g := svg.NewGroup(cg)
	rs.gpStack.Push(g)
	return g
}

func (rs *Renderer) RenderPath(pt *render.Path) {
	p := pt.Path
	pc := &pt.Context
	cg := rs.gpStack.Peek()
	sp := svg.NewPath(cg)
	sp.Data = p.Clone()
	props := map[string]any{}
	pt.Context.Style.GetProperties(props)
	if !pc.Transform.IsIdentity() {
		props["transform"] = pc.Transform.String()
	}
	sp.Properties = props
	// rs.Scanner.SetClip(pc.Bounds.Rect.ToRect())
}

func (rs *Renderer) PushContext(pt *render.ContextPush) {
	pc := &pt.Context
	g := rs.PushGroup()
	g.Paint.Transform = pc.Transform
}

func (rs *Renderer) PopContext(pt *render.ContextPop) {
	rs.gpStack.Pop()
}

func (rs *Renderer) RenderText(pt *render.Text) {
	pc := &pt.Context
	cg := rs.gpStack.Peek()
	tg := svg.NewGroup(cg)
	props := map[string]any{}
	pt.Context.Style.GetProperties(props)
	if !pc.Transform.IsIdentity() {
		props["transform"] = pc.Transform.String()
	}
	pos := pt.Position
	tx := pt.Text.Source
	txt := tx.Join()
	for li := range pt.Text.Lines {
		ln := &pt.Text.Lines[li]
		lpos := pos.Add(ln.Offset)
		rpos := lpos
		for ri := range ln.Runs {
			run := ln.Runs[ri].(*shapedgt.Run)
			rs := run.Runes().Start
			re := run.Runes().End
			si, _, _ := tx.Index(rs)
			sty, _ := tx.Span(si)
			rtxt := txt[rs:re]

			st := svg.NewText(tg)
			st.Text = string(rtxt)
			rprops := maps.Clone(props)
			if pc.Style.UnitContext.DPI != 160 {
				sty.Size *= pc.Style.UnitContext.DPI / 160
			}
			pt.Context.Style.Text.ToProperties(sty, rprops)
			rprops["x"] = reflectx.ToString(rpos.X)
			rprops["y"] = reflectx.ToString(rpos.Y)
			st.Pos = rpos
			st.Properties = rprops

			rpos.X += run.Advance()
		}
	}
}

func (rs *Renderer) RenderImage(pr *pimage.Params) {
	usrc := imagex.Unwrap(pr.Source)
	umask := imagex.Unwrap(pr.Mask)
	cg := rs.gpStack.Peek()

	nilSrc := usrc == nil
	if r, ok := usrc.(*image.RGBA); ok && r == nil {
		nilSrc = true
	}
	if pr.Rect == (image.Rectangle{}) {
		pr.Rect = image.Rectangle{Max: rs.size.ToPoint()}
	}

	// todo: handle masks!

	// Fast path for [image.Uniform]
	if u, ok := usrc.(*image.Uniform); nilSrc || ok && umask == nil {
		r := svg.NewRect(cg)
		r.Pos = math32.FromPoint(pr.Rect.Min)
		r.Size = math32.FromPoint(pr.Rect.Size())
		r.SetProperty("fill", colors.AsHex(u.C))
		return
	}

	if gr, ok := usrc.(gradient.Gradient); ok {
		_ = gr
		// todo: handle:
		return
	}

	sz := pr.Rect.Size()

	simg := svg.NewImage(cg)
	simg.SetImage(usrc, float32(sz.X), float32(sz.Y))
	simg.Pos = math32.FromPoint(pr.Rect.Min)
	// todo: ViewBox?
}
