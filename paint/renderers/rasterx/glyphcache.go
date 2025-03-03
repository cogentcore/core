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
	TheGlyphCache GlyphCache

	// UseGlyphCache determines if the glyph cache is used.
	UseGlyphCache = true
)

func init() {
	TheGlyphCache.Init()
}

// GlyphCache holds cached rendered font glyphs.
type GlyphCache struct {
	// Faces is a map of faces.
	Faces map[*font.Face]*FaceCache

	// rendering:
	MaxSize    image.Point
	image      *image.RGBA
	scanner    *scan.Scanner
	imgSpanner *scan.ImgSpanner
	filler     *Filler
	sync.Mutex
}

type sizeGID struct {
	size image.Point
	gid  font.GID
}

// FaceCache holds the cached glyphs for given face.
type FaceCache struct {
	Face   *font.Face
	Glyphs map[sizeGID]*image.Alpha
}

func (fc *GlyphCache) Init() {
	fc.Faces = make(map[*font.Face]*FaceCache)
	fc.MaxSize = image.Point{30, 30}
	sz := fc.MaxSize
	fc.image = image.NewRGBA(image.Rectangle{Max: sz})
	fc.imgSpanner = scan.NewImgSpanner(fc.image)
	fc.scanner = scan.NewScanner(fc.imgSpanner, sz.X, sz.Y)
	fc.filler = NewFiller(sz.X, sz.Y, fc.scanner)
	fc.filler.SetWinding(true)
	fc.filler.SetColor(colors.Uniform(color.Black))
}

// Glyph returns an existing cached glyph or a newly rendered one.
func (gc *GlyphCache) Glyph(face *font.Face, g *shaping.Glyph, outline font.GlyphOutline, scale float32) *image.Alpha {
	gc.Lock()
	defer gc.Unlock()

	// fmt.Printf("g: %#v\n", g)
	fsize := image.Point{X: int(g.Width.Ceil()), Y: -int(g.Height.Ceil())}
	size := fsize.Add(image.Point{8, 8})
	if size.X > gc.MaxSize.X || size.Y > gc.MaxSize.Y {
		return nil
	}
	szgid := sizeGID{size: fsize, gid: g.GlyphID}

	fc, hasfc := gc.Faces[face]
	if hasfc {
		mask := fc.Glyphs[szgid]
		if mask != nil {
			return mask
		}
	}
	mask := gc.renderGlyph(face, g.GlyphID, g, outline, size, scale)
	if !hasfc {
		fc = &FaceCache{Face: face}
		fc.Glyphs = make(map[sizeGID]*image.Alpha)
	}
	fc.Glyphs[szgid] = mask
	gc.Faces[face] = fc
	return mask
}

// renderGlyph renders the given glyph and caches the result.
func (gc *GlyphCache) renderGlyph(face *font.Face, gid font.GID, g *shaping.Glyph, outline font.GlyphOutline, size image.Point, scale float32) *image.Alpha {
	// clear target:
	draw.Draw(gc.image, gc.image.Bounds(), colors.Uniform(color.Transparent), image.Point{0, 0}, draw.Src)

	pos := math32.Vec2(math32.FromFixed(g.XOffset)+math32.FromFixed(g.XBearing), -math32.FromFixed(g.YOffset)+math32.FromFixed(g.YBearing))
	// fmt.Println(pos)
	x := pos.X + 4
	y := pos.Y + 4
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
