// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svgrender

import (
	"bytes"
	"image"

	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/svg"
)

// Renderer is the SVG renderer.
type Renderer struct {
	size    math32.Vector2
	SVG     *svg.SVG
	gpStack stack.Stack[*svg.Group]
}

func New(size math32.Vector2) render.Renderer {
	rs := &Renderer{}
	rs.SetSize(units.UnitDot, size)
	return rs
}

func (rs *Renderer) Image() image.Image { return rs.SVG.RenderImage() }

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
func (rs *Renderer) Render(r render.Render) {
	rs.SVG = svg.NewSVG(int(rs.size.X), int(rs.size.Y))
	rs.SVG.PhysicalWidth.Dot(rs.size.X)
	rs.SVG.PhysicalHeight.Dot(rs.size.Y)
	ps := styles.NewPaint()
	ps.Defaults()
	rs.SVG.SetUnitContext(ps, rs.size, rs.size)
	rs.gpStack = nil
	bg := svg.NewGroup(rs.SVG.Root)
	rs.gpStack.Push(bg)
	for _, ri := range r {
		switch x := ri.(type) {
		case *render.Path:
			rs.RenderPath(x)
		case *pimage.Params:
			// x.Render(rs.image)
		case *render.Text:
			// rs.RenderText(x)
		case *render.ContextPush:
			rs.PushContext(x)
		case *render.ContextPop:
			rs.PopContext(x)
		}
	}
	rs.SVG.Render()
}

func (rs *Renderer) PushGroup() *svg.Group {
	cg := rs.gpStack.Peek()
	g := svg.NewGroup(cg)
	rs.gpStack.Push(g)
	return g
}

func (rs *Renderer) NewPath() *svg.Path {
	cg := rs.gpStack.Peek()
	p := svg.NewPath(cg)
	return p
}

func (rs *Renderer) RenderPath(pt *render.Path) {
	p := pt.Path
	pc := &pt.Context
	sp := rs.NewPath()
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
