// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
)

// Pattern is an SVG pattern element: the fill or stroke of the elements
// that refer to it (as fill="url(#name)") is its children, rendered in a
// tile that repeats. It is not rendered itself.
type Pattern struct {
	NodeBase

	// ViewBox is the view box of the pattern, if any.
	ViewBox ViewBox
}

func (g *Pattern) SVGName() string { return "pattern" }

func (g *Pattern) EnforceSVGName() bool { return false }

func (g *Pattern) Render(sv *SVG) {} // only for the elements that refer to it

// maxPatternTile is the maximum width and height of the image of the tile
// of a pattern, in pixels.
const maxPatternTile = 2048

// attr returns the value of the given attribute (property) of the pattern
// or of the patterns that it refers to (with href), and whether it has it.
func (g *Pattern) attr(sv *SVG, name string) (string, bool) {
	for p, n := g, 0; p != nil && n < 10; n++ {
		if v, ok := p.Properties[name]; ok {
			return strings.TrimSpace(fmt.Sprint(v)), true
		}
		p = p.ref(sv)
	}
	return "", false
}

// ref returns the pattern that the pattern refers to with href, if any.
func (g *Pattern) ref(sv *SVG) *Pattern {
	href, ok := g.Properties["href"]
	if !ok {
		return nil
	}
	name := strings.TrimPrefix(strings.TrimSpace(fmt.Sprint(href)), "#")
	p, _ := sv.findByName(name).(*Pattern)
	if p == g {
		return nil
	}
	return p
}

// content returns the pattern with the children of the pattern (its own
// or of the patterns that it refers to), and the one with its view box.
func (g *Pattern) content(sv *SVG) (kids, vb *Pattern) {
	for p, n := g, 0; p != nil && n < 10; n++ {
		if kids == nil && len(p.Children) > 0 {
			kids = p
		}
		if vb == nil && p.ViewBox.Size.X > 0 && p.ViewBox.Size.Y > 0 {
			vb = p
		}
		p = p.ref(sv)
	}
	return
}

// length returns the value of the given attribute of the pattern as a
// length: in the units of the given size for objectBoundingBox units (a
// fraction or percent), or in user units (or percent of the given size).
func (g *Pattern) length(sv *SVG, name string, size float32) float32 {
	v, ok := g.attr(sv, name)
	if !ok {
		return 0
	}
	if pct, ok := strings.CutSuffix(v, "%"); ok {
		f, _ := math32.ParseFloat32(pct)
		return f / 100 * size
	}
	f, _ := math32.ParseFloat32(strings.TrimSuffix(v, "px"))
	return f
}

// preparePatterns makes the pattern fill and stroke of the node, if any,
// ready to render, for the node, with the given transform from the user
// space of the current context to that of the node.
func (g *NodeBase) preparePatterns(sv *SVG, xf math32.Matrix2) {
	for _, clr := range []*image.Image{&g.Paint.Fill.Color, &g.Paint.Stroke.Color} {
		pf, ok := (*clr).(*patternFill)
		if !ok {
			continue
		}
		npf := &patternFill{pattern: pf.pattern}
		npf.prepare(sv, g, xf)
		*clr = npf // for the node only
	}
}

// patternFill is the fill (or stroke) of an element with a [Pattern]: the
// image of its tile, repeated. It is a [gradient.Gradient] so that the
// renderers update it (with [patternFill.Update]) before they render it.
type patternFill struct {
	base    gradient.Base
	pattern *Pattern

	// items render the tile, in the pixels of its image.
	items render.Render

	// size is the size of the image of the tile, in pixels.
	size image.Point

	// toTile transforms the pixels where the pattern is rendered to the
	// pixels of the image of the tile (before they wrap around).
	toTile math32.Matrix2

	once    sync.Once
	tile    *image.RGBA
	opacity float32
}

// prepare renders the items of the tile of the pattern for the given
// node, with the given transform from the user space of the current
// context to that of the node.
func (pf *patternFill) prepare(sv *SVG, target *NodeBase, xf math32.Matrix2) {
	g := pf.pattern
	pf.opacity = 1
	bb := sv.userBBox(target)
	bsz := bb.Size()
	obb := true // patternUnits
	if u, ok := g.attr(sv, "patternUnits"); ok {
		obb = u != "userSpaceOnUse"
	}
	var tile math32.Box2
	if obb {
		tile.Min.Set(bb.Min.X+g.length(sv, "x", 1)*bsz.X, bb.Min.Y+g.length(sv, "y", 1)*bsz.Y)
		tile.Max = tile.Min.Add(math32.Vec2(g.length(sv, "width", 1)*bsz.X, g.length(sv, "height", 1)*bsz.Y))
	} else {
		vp := sv.Root.ViewBox.Size
		tile.Min.Set(g.length(sv, "x", vp.X), g.length(sv, "y", vp.Y))
		tile.Max = tile.Min.Add(math32.Vec2(g.length(sv, "width", vp.X), g.length(sv, "height", vp.Y)))
	}
	tsz := tile.Size()
	if tsz.X <= 0 || tsz.Y <= 0 {
		return // nothing is rendered
	}
	ptf := math32.Identity2()
	if v, ok := g.attr(sv, "patternTransform"); ok {
		ptf.SetString(v)
	}
	// from pattern space to the pixels where the node renders
	toPixels := sv.painter.Context().Cumulative.Mul(xf).Mul(ptf)
	sx := math32.Vec2(toPixels.XX, toPixels.YX).Length()
	sy := math32.Vec2(toPixels.XY, toPixels.YY).Length()
	pf.size = image.Pt(int(math32.Ceil(tsz.X*sx)), int(math32.Ceil(tsz.Y*sy)))
	pf.size.X = min(max(pf.size.X, 1), maxPatternTile)
	pf.size.Y = min(max(pf.size.Y, 1), maxPatternTile)
	scale := math32.Vec2(float32(pf.size.X)/tsz.X, float32(pf.size.Y)/tsz.Y)
	// tile pixels from pattern space
	tileXf := math32.Scale2D(scale.X, scale.Y).Translate(-tile.Min.X, -tile.Min.Y)
	pf.toTile = tileXf.Mul(toPixels.Inverse())

	// the content, from its units (or view box) to the tile
	contentXf := math32.Translate2D(tile.Min.X, tile.Min.Y)
	kids, vbp := g.content(sv)
	if vbp != nil {
		vb := vbp.ViewBox
		_, trans, vscale := vb.Transform(tsz)
		trans.SetSub(vb.Min)
		contentXf = contentXf.Mul(math32.Scale2D(vscale.X, vscale.Y).Translate(trans.X, trans.Y))
	} else if u, _ := g.attr(sv, "patternContentUnits"); u == "objectBoundingBox" {
		contentXf = math32.Translate2D(tile.Min.X, tile.Min.Y).Scale(bsz.X, bsz.Y)
	}
	if kids == nil {
		return
	}
	pc := sv.painter
	cur := pc.Context().Cumulative
	pf.items = pc.RenderSeparately(func() {
		sty := styles.NewPaint()
		sty.UnitContext = pc.Context().Style.UnitContext
		sty.Transform = cur.Inverse().Mul(tileXf.Mul(contentXf))
		pc.PushContext(sty, render.NewBounds(0, 0, float32(pf.size.X), float32(pf.size.Y), sides.Floats{}))
		for _, k := range kids.Children {
			nb := k.(Node).AsNodeBase()
			def := nb.isDef
			nb.isDef = true // rendered whatever their visible box
			k.(Node).Render(sv)
			nb.isDef = def
		}
		pc.PopContext()
	})
}

func (pf *patternFill) AsBase() *gradient.Base { return &pf.base }

// CopyGradient returns a copy of the pattern fill, which shares its tile.
func (pf *patternFill) CopyGradient() gradient.Gradient {
	pf.once.Do(func() { pf.renderTile() })
	cp := &patternFill{pattern: pf.pattern, items: pf.items, size: pf.size, toTile: pf.toTile, tile: pf.tile, opacity: pf.opacity}
	cp.once.Do(func() {}) // the tile is shared
	cp.base.ApplyFuncs = pf.base.ApplyFuncs.Clone()
	return cp
}

// Update sets the opacity of the pattern, before it is rendered.
func (pf *patternFill) Update(opacity float32, box math32.Box2, objTransform math32.Matrix2) {
	pf.opacity = opacity
}

func (pf *patternFill) ColorModel() color.Model { return color.RGBAModel }

func (pf *patternFill) Bounds() image.Rectangle {
	return image.Rect(-1e9, -1e9, 1e9, 1e9)
}

// At returns the color of the pattern at the given pixel: of the pixel
// of the tile that it is at.
func (pf *patternFill) At(x, y int) color.Color {
	pf.once.Do(func() { pf.renderTile() })
	if pf.tile == nil {
		return color.RGBA{}
	}
	p := pf.toTile.MulPoint(math32.Vec2(float32(x)+0.5, float32(y)+0.5))
	tx := int(math32.Floor(p.X)) % pf.size.X
	ty := int(math32.Floor(p.Y)) % pf.size.Y
	if tx < 0 {
		tx += pf.size.X
	}
	if ty < 0 {
		ty += pf.size.Y
	}
	c := pf.tile.RGBAAt(tx, ty)
	if pf.opacity < 1 {
		o := pf.opacity
		c = color.RGBA{uint8(float32(c.R) * o), uint8(float32(c.G) * o), uint8(float32(c.B) * o), uint8(float32(c.A) * o)}
	}
	if len(pf.base.ApplyFuncs) > 0 {
		return pf.base.ApplyFuncs.Apply(c)
	}
	return c
}

// renderTile renders the image of the tile of the pattern.
func (pf *patternFill) renderTile() {
	if len(pf.items) == 0 || paint.NewImageRenderer == nil {
		return
	}
	img := paint.NewImageRenderer(math32.FromPoint(pf.size)).Render(pf.items).Image()
	pf.tile, _ = img.(*image.RGBA)
}
