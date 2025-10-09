// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pdfrender

import (
	"bytes"
	"image"
	"io"
	"strconv"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/pdf"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles/units"
)

// Renderer is the PDF renderer.
type Renderer struct {
	size        math32.Vector2
	sizeUnits   units.Units
	unitContext units.Context

	PDF *pdf.PDF

	buff *bytes.Buffer

	// lyStack is a stack of layers used while building the pdf (int layer id)
	lyStack stack.Stack[int]
}

func New(size math32.Vector2, un *units.Context) render.Renderer {
	rs := &Renderer{unitContext: *un}
	rs.SetSize(units.UnitDot, size)
	return rs
}

func (rs *Renderer) Image() image.Image {
	return nil // can't generate an image
}

func (rs *Renderer) Source() []byte {
	if rs.buff == nil {
		return nil
	}
	return rs.buff.Bytes()
}

func (rs *Renderer) Size() (units.Units, math32.Vector2) {
	return rs.sizeUnits, rs.size
}

func (rs *Renderer) SetSize(un units.Units, size math32.Vector2) {
	if rs.sizeUnits == un && rs.size == size {
		return
	}
	rs.sizeUnits = un
	rs.size = size
}

// Render is the main rendering function.
func (rs *Renderer) Render(r render.Render) render.Renderer {
	rs.buff = &bytes.Buffer{}
	rs.StartRender(rs.buff)
	rs.RenderPage(r)
	rs.EndRender()
	return rs
}

// StartRender creates the renderer.
func (rs *Renderer) StartRender(w io.Writer) {
	sx := rs.unitContext.Convert(float32(rs.size.X), rs.sizeUnits, units.UnitPt)
	sy := rs.unitContext.Convert(float32(rs.size.Y), rs.sizeUnits, units.UnitPt)
	rs.PDF = pdf.New(w, sx, sy, &rs.unitContext)
	rs.lyStack = nil
}

// EndRender finishes the render
func (rs *Renderer) EndRender() {
	rs.PDF.Close()
}

// AddPage adds a new page of the same size.
func (rs *Renderer) AddPage() {
	sx := rs.unitContext.Convert(float32(rs.size.X), rs.sizeUnits, units.UnitPt)
	sy := rs.unitContext.Convert(float32(rs.size.Y), rs.sizeUnits, units.UnitPt)
	rs.PDF.NewPage(sx, sy)
}

// RenderPage renders the content to current PDF page
func (rs *Renderer) RenderPage(r render.Render) {
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
}

func (rs *Renderer) PushLayer(m math32.Matrix2) int {
	cg := rs.lyStack.Peek()
	nm := strconv.Itoa(cg + 1)
	g := rs.PDF.AddLayer(nm, true)
	rs.PDF.BeginLayer(g, m)
	rs.lyStack.Push(g)
	return g
}

func (rs *Renderer) PopLayer() int {
	cg := rs.lyStack.Pop()
	rs.PDF.EndLayer()
	return cg
}

func (rs *Renderer) RenderPath(pt *render.Path) {
	p := pt.Path
	pc := &pt.Context
	rs.PDF.Path(p, &pc.Style, pc.Transform)
}

func (rs *Renderer) PushContext(pt *render.ContextPush) {
	rs.PushLayer(pt.Context.Transform)
}

func (rs *Renderer) PopContext(pt *render.ContextPop) {
	rs.PopLayer()
}

func (rs *Renderer) RenderText(pt *render.Text) {
	pc := &pt.Context
	rs.PDF.Text(&pc.Style, pc.Transform, pt.Position, pt.Text)
}

func (rs *Renderer) RenderImage(pr *pimage.Params) {
	usrc := imagex.Unwrap(pr.Source)
	umask := imagex.Unwrap(pr.Mask)

	nilSrc := usrc == nil
	if r, ok := usrc.(*image.RGBA); ok && r == nil {
		nilSrc = true
	}
	if pr.Rect == (image.Rectangle{}) {
		pr.Rect = image.Rectangle{Max: rs.size.ToPoint()}
	}

	if pr.Anchor != "" {
		rs.PDF.AddAnchor(pr.Anchor, math32.FromPoint(pr.Rect.Min))
	}

	// todo: handle masks!

	// Fast path for [image.Uniform]
	if u, ok := usrc.(*image.Uniform); nilSrc || ok && umask == nil {
		rs.PDF.FillBox(math32.Identity2(), math32.B2FromRect(pr.Rect), u)
		return
	}

	if gr, ok := usrc.(gradient.Gradient); ok {
		_ = gr
		// todo: handle:
		return
	}

	// sz := pr.Rect.Size()
	m := math32.Translate2D(float32(pr.Rect.Min.X), float32(pr.Rect.Min.Y))
	rs.PDF.Image(usrc, m)
	// simg.Pos = math32.FromPoint(pr.Rect.Min)
}
