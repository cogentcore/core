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

func (rs *Renderer) writePath(pt *render.Path) {
	rs.ctx.Call("beginPath")
	for scanner := pt.Path.Scanner(); scanner.Scan(); {
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

	strokeUnsupported := false
	// if m.IsSimilarity() { // TODO: implement
	if true {
		scale := math32.Sqrt(math32.Abs(pt.Context.Transform.Det()))
		// TODO: this is a hack to get the effect of [ppath.VectorEffectNonScalingStroke]
		style.Stroke.Width.Dots /= scale
		// style.Stroke.DashOffset, style.Stroke.Dashes = ppath.ScaleDash(style.Stroke.Width.Dots, style.Stroke.DashOffset, style.Stroke.Dashes)
	} else {
		strokeUnsupported = true
	}

	if style.HasFill() || (style.HasStroke() && !strokeUnsupported) {
		rs.writePath(pt)
	}

	if style.HasFill() {
		rs.setFill(style.Fill.Color)
		rule := "nonzero"
		if style.Fill.Rule == ppath.EvenOdd {
			rule = "evenodd"
		}
		rs.ctx.Call("fill", rule)
	}
	if style.HasStroke() && !strokeUnsupported {
		rs.setStroke(&style.Stroke)
		rs.ctx.Call("stroke")
	} else if style.HasStroke() {
		// stroke settings unsupported by HTML Canvas, draw stroke explicitly
		// TODO: check when this is happening, maybe remove or use rasterx?
		if len(style.Stroke.Dashes) > 0 {
			pt.Path = pt.Path.Dash(style.Stroke.DashOffset, style.Stroke.Dashes...)
		}
		pt.Path = pt.Path.Stroke(style.Stroke.Width.Dots, ppath.CapFromStyle(style.Stroke.Cap), ppath.JoinFromStyle(style.Stroke.Join), 1)
		rs.writePath(pt)
		rs.ctx.Set("fillStyle", rs.imageToStyle(style.Stroke.Color))
		rs.ctx.Call("fill")
	}
}
