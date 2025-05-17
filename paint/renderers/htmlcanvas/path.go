// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

//go:build js

package htmlcanvas

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/render"
)

func (rs *Renderer) writePath(pt *ppath.Path) {
	rs.ctx.Call("beginPath")
	for scanner := pt.Scanner(); scanner.Scan(); {
		end := scanner.End()
		switch scanner.Cmd() {
		case ppath.MoveTo:
			rs.ctx.Call("moveTo", end.X, end.Y)
		case ppath.LineTo:
			rs.ctx.Call("lineTo", end.X, end.Y)
		case ppath.QuadTo:
			cp := scanner.CP1()
			rs.ctx.Call("quadraticCurveTo", cp.X, cp.Y, end.X, end.Y)
		case ppath.CubeTo:
			cp1, cp2 := scanner.CP1(), scanner.CP2()
			rs.ctx.Call("bezierCurveTo", cp1.X, cp1.Y, cp2.X, cp2.Y, end.X, end.Y)
		case ppath.Close:
			rs.ctx.Call("closePath")
		}
	}
}

func (rs *Renderer) RenderPath(pt *render.Path) {
	if pt.Path.Empty() {
		return
	}
	style := &pt.Context.Style
	p := pt.Path
	if !ppath.ArcToCubeImmediate {
		p = p.ReplaceArcs() // TODO: should we do this in writePath?
	}
	rs.setTransform(&pt.Context)

	if style.HasFill() || style.HasStroke() {
		rs.writePath(&pt.Path)
	}

	rs.curRect = pt.Path.FastBounds().ToRect() // TODO: more performance optimized approach (such as only computing for gradients)?
	if style.HasFill() {
		rs.setFill(style.Fill.Color)
		rs.ctx.Call("fill", style.Fill.Rule.String())
	}
	if style.HasStroke() {
		scale := math32.Sqrt(math32.Abs(pt.Context.Transform.Det()))
		// note: this is a hack to get the effect of [ppath.VectorEffectNonScalingStroke]
		style.Stroke.Width.Dots /= scale
		rs.setStroke(&style.Stroke)
		rs.ctx.Call("stroke")
	}
}
