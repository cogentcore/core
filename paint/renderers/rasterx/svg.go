// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx

import (
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
)

func (rs *Renderer) SVG(ctx *render.Context, run *shapedgt.Run, svgCmds string, bb math32.Box2, pos math32.Vector2, identity bool) {
	scale := math32.FromFixed(run.Size) / float32(run.Face.Upem())
	tx := ctx.Transform.Scale(scale, scale)
	paths := strings.Split(svgCmds, "<path d=")
	for i, p := range paths {
		if i == 0 {
			continue
		}
		d := p[1:]
		eq := strings.LastIndex(d, `"`)
		d = d[:eq]
		eq = strings.LastIndex(d, `"`)
		fill := ""
		if eq > 0 {
			fill = d[eq+1:]
			d = d[:eq]
			eq = strings.LastIndex(d, `"`)
			d = d[:eq]
			fi := strings.Index(fill, `fill="`)
			if fi >= 0 {
				fill = fill[fi+6:]
			}
			// fmt.Println("path d=\n", d, "\nfill=", fill)
		}
		pp, _ := ppath.ParseSVGPath(d)
		rs.Path.Clear()
		PathToRasterx(&rs.Path, pp, tx, pos)
		rs.Path.Stop(true)
		rf := &rs.Raster.Filler
		rf.SetWinding(true)
		clr := errors.Log1(colors.FromHex(fill[1:]))
		rf.SetColor(colors.Uniform(clr))
		rs.Path.AddTo(rf)
		rf.Draw()
		rf.Clear()
	}

	rs.Path.Clear()
}
