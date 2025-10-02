// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pdfrender

import (
	"bytes"
	"image"
	"maps"
	"strconv"

	"codeberg.org/go-pdf/fpdf"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
)

// Renderer is the PDF renderer.
type Renderer struct {
	size math32.Vector2

	PDF *fpdf.Fpdf

	// lyStack is a stack of layers used while building the pdf (int layer id)
	lyStack stack.Stack[int]
}

func New(size math32.Vector2) render.Renderer {
	rs := &Renderer{}
	rs.SetSize(units.UnitDot, size)
	return rs
}

func (rs *Renderer) Image() image.Image {
	if rs.PDF == nil {
		return nil
	}
	pc := rs.FPDf.Render(nil)
	ir := paint.NewImageRenderer(rs.size)
	ir.Render(pc.RenderDone())
	return ir.Image()
}

func (rs *Renderer) Source() []byte {
	if rs.PDF == nil {
		return nil
	}
	var b bytes.Buffer
	rs.FPDf.WriteXML(&b, true)
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
	rs.PDF = fpdf.New("P", "mm", core.SystemSettings.PageSize.String(), ".")
	rs.PDF.SetFont("Arial", "", 16)
	rs.lyStack = nil
	bg := rs.PDF.AddLayer("bg", true)
	rs.PDF.BeginLayer(bg)
	rs.lyStack.Push(bg)
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
	rs.PDF.EndLayer()
	return rs
}

func (rs *Renderer) PushLayer() int {
	cg := rs.lyStack.Peek()
	nm := strconv.Itoa(cg + 1)
	g := rs.PDF.AddLayer(nm)
	rs.PDF.BeginLayer(g)
	rs.lyStack.Push(g)
	return g
}

func (rs *Renderer) PopLayer() int {
	cg := rs.lyStack.Pop()
	rs.PDF.EndLayer()
}

func (rs *Renderer) RenderPath(pt *render.Path) {
	p := pt.Path
	pc := &pt.Context

	closed := false
	data := p.Copy().Transform(pc.Transform).ToPDF()
	if 1 < len(data) && data[len(data)-1] == 'h' {
		data = data[:len(data)-2]
		closed = true
	}

	cg := rs.lyStack.Peek()
	sp := fpdf.NewPath(cg)
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
	g := rs.PushLayer()
	g.Paint.Transform = pc.Transform
}

func (rs *Renderer) PopContext(pt *render.ContextPop) {
	rs.PopLayer()
}

func (rs *Renderer) RenderText(pt *render.Text) {
	pc := &pt.Context
	cg := rs.lyStack.Peek()
	tg := fpdf.NewLayer(cg)
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

			st := fpdf.NewText(tg)
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
	cg := rs.lyStack.Peek()

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
		r := fpdf.NewRect(cg)
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

	simg := fpdf.NewImage(cg)
	simg.SetImage(usrc, float32(sz.X), float32(sz.Y))
	simg.Pos = math32.FromPoint(pr.Rect.Min)
	// todo: ViewBox?
}
