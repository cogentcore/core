// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/renderers/rasterx/scan"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/shaping"
)

var (
	// TheGlyphCache is the shared font glyph bitmap render cache.
	theGlyphCache glyphCache

	// UseGlyphCache determines if the glyph cache is used.
	UseGlyphCache = true
)

const (
	// glyphMaxSize is the max size in either dim for the render mask.
	glyphMaxSize = 128

	// glyphMaskBorder is the extra amount on each side to include around the glyph bounds.
	glyphMaskBorder = 2

	// glyphMaskOffsets is the number of different subpixel offsets to render, in each axis.
	// The memory usage goes as the square of this number, and 4 produces very good results,
	// while 2 is acceptable, and is significantly better than 1. 8 is overkill.
	glyphMaskOffsets = 4
)

func init() {
	theGlyphCache.init()
}

// GlyphCache holds cached rendered font glyphs.
type glyphCache struct {
	glyphs     map[*font.Face]map[glyphKey]*image.Alpha
	maxSize    image.Point
	image      *image.RGBA
	scanner    *scan.Scanner
	imgSpanner *scan.ImgSpanner
	filler     *Filler
	sync.Mutex
}

// glyphKey is the key for encoding a mask render.
type glyphKey struct {
	gid font.GID // uint32
	sx  uint8    // size
	sy  uint8
	ox  uint8 // offset
	oy  uint8
}

func (fc *glyphCache) init() {
	fc.glyphs = make(map[*font.Face]map[glyphKey]*image.Alpha)
	fc.maxSize = image.Point{glyphMaxSize, glyphMaxSize}
	sz := fc.maxSize
	fc.image = image.NewRGBA(image.Rectangle{Max: sz})
	fc.imgSpanner = scan.NewImgSpanner(fc.image)
	fc.scanner = scan.NewScanner(fc.imgSpanner, sz.X, sz.Y)
	fc.filler = NewFiller(sz.X, sz.Y, fc.scanner)
	fc.filler.SetWinding(true)
	fc.filler.SetColor(colors.Uniform(color.Black))
	fc.scanner.SetClip(fc.image.Bounds())
}

// Glyph returns an existing cached glyph or a newly rendered one,
// and the top-left rendering position to use, based on pos arg.
// fractional offsets are supported to improve quality.
func (gc *glyphCache) Glyph(face *font.Face, g *shaping.Glyph, outline font.GlyphOutline, scale float32, pos math32.Vector2) (*image.Alpha, image.Point) {
	gc.Lock()
	defer gc.Unlock()

	fsize := image.Point{X: int(g.Width.Ceil()), Y: -int(g.Height.Ceil())}
	size := fsize.Add(image.Point{2 * glyphMaskBorder, 2 * glyphMaskBorder})
	if size.X <= 0 || size.X > glyphMaxSize || size.Y <= 0 || size.Y > glyphMaxSize {
		return nil, image.Point{}
	}
	// fmt.Println(face.Describe().Family, g.GlyphID, "wd, ht:", math32.FromFixed(g.Width), -math32.FromFixed(g.Height), "size:", size)
	// fmt.Printf("g: %#v\n", g)

	pf := pos.Floor()
	pi := pf.ToPoint().Sub(image.Point{glyphMaskBorder, glyphMaskBorder})
	pi.X += g.XBearing.Round()
	pi.Y -= g.YBearing.Round()
	off := pos.Sub(pf)
	oi := off.MulScalar(glyphMaskOffsets).Floor().ToPoint()
	// fmt.Println("pos:", pos, "oi:", oi, "pi:", pi)

	key := glyphKey{gid: g.GlyphID, sx: uint8(fsize.X), sy: uint8(fsize.Y), ox: uint8(oi.X), oy: uint8(oi.Y)}

	fc, hasfc := gc.glyphs[face]
	if hasfc {
		mask := fc[key]
		if mask != nil {
			return mask, pi
		}
	} else {
		fc = make(map[glyphKey]*image.Alpha)
	}

	mask := gc.renderGlyph(face, g.GlyphID, g, outline, size, scale, oi.X, oi.Y)
	fc[key] = mask
	gc.glyphs[face] = fc
	// fmt.Println(gc.CacheSize())
	return mask, pi
}

// renderGlyph renders the given glyph and caches the result.
func (gc *glyphCache) renderGlyph(face *font.Face, gid font.GID, g *shaping.Glyph, outline font.GlyphOutline, size image.Point, scale float32, xo, yo int) *image.Alpha {
	// clear target:
	draw.Draw(gc.image, gc.image.Bounds(), colors.Uniform(color.Transparent), image.Point{0, 0}, draw.Src)

	od := float32(1) / glyphMaskOffsets
	x := -float32(g.XBearing.Round()) + float32(xo)*od + glyphMaskBorder
	y := float32(g.YBearing.Round()) + float32(yo)*od + glyphMaskBorder
	rs := gc.filler
	rs.Clear()
	for _, s := range outline.Segments {
		p0 := math32.Vec2(s.Args[0].X*scale+x, -s.Args[0].Y*scale+y)
		switch s.Op {
		case opentype.SegmentOpMoveTo:
			rs.Start(p0.ToFixed())
		case opentype.SegmentOpLineTo:
			rs.Line(p0.ToFixed())
		case opentype.SegmentOpQuadTo:
			p1 := math32.Vec2(s.Args[1].X*scale+x, -s.Args[1].Y*scale+y)
			rs.QuadBezier(p0.ToFixed(), p1.ToFixed())
		case opentype.SegmentOpCubeTo:
			p1 := math32.Vec2(s.Args[1].X*scale+x, -s.Args[1].Y*scale+y)
			p2 := math32.Vec2(s.Args[2].X*scale+x, -s.Args[2].Y*scale+y)
			rs.CubeBezier(p0.ToFixed(), p1.ToFixed(), p2.ToFixed())
		}
	}
	rs.Stop(true)
	rs.Draw()
	rs.Clear()
	bb := image.Rectangle{Max: size}

	mask := image.NewAlpha(bb)
	draw.Draw(mask, bb, gc.image, image.Point{}, draw.Src)
	// fmt.Println("size:", size, *mask)
	// fmt.Println("render:", gid, size)
	return mask
}

// CacheSize reports the total number of bytes used for image masks.
// For example, the cogent core docs took about 3.5mb using 4
func (gc *glyphCache) CacheSize() int {
	gc.Lock()
	defer gc.Unlock()
	total := 0
	for _, fc := range gc.glyphs {
		for _, mask := range fc {
			sz := mask.Bounds().Size()
			total += sz.X * sz.Y
		}
	}
	return total
}
