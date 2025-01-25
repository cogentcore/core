// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/linebender/vello/blob/main/vello_encoding/src/path.rs

package rasterg

import (
	"math"
	"unsafe"

	"cogentcore.org/core/styles"
)

type Style struct {
	flagsAndMiterLimit uint32
	lineWidth          float32
}

const (
	FLAGS_STYLE_BIT             uint32 = 0x80000000
	FLAGS_FILL_BIT              uint32 = 0x40000000
	FLAGS_JOIN_BITS_BEVEL       uint32 = 0
	FLAGS_JOIN_BITS_MITER       uint32 = 0x10000000
	FLAGS_JOIN_BITS_ROUND       uint32 = 0x20000000
	FLAGS_JOIN_MASK             uint32 = 0x30000000
	FLAGS_CAP_BITS_BUTT         uint32 = 0
	FLAGS_CAP_BITS_SQUARE       uint32 = 0x01000000
	FLAGS_CAP_BITS_ROUND        uint32 = 0x02000000
	FLAGS_START_CAP_BITS_BUTT   uint32 = FLAGS_CAP_BITS_BUTT << 2
	FLAGS_START_CAP_BITS_SQUARE uint32 = FLAGS_CAP_BITS_SQUARE << 2
	FLAGS_START_CAP_BITS_ROUND  uint32 = FLAGS_CAP_BITS_ROUND << 2
	FLAGS_END_CAP_BITS_BUTT     uint32 = FLAGS_CAP_BITS_BUTT
	FLAGS_END_CAP_BITS_SQUARE   uint32 = FLAGS_CAP_BITS_SQUARE
	FLAGS_END_CAP_BITS_ROUND    uint32 = FLAGS_CAP_BITS_ROUND
	FLAGS_START_CAP_MASK        uint32 = 0x0C000000
	FLAGS_END_CAP_MASK          uint32 = 0x03000000
	MITER_LIMIT_MASK            uint32 = 0xFFFF
)

func NewStyleFromFill(fill styles.FillRules) Style {
	var fillBit uint32
	if fill == styles.FillRuleEvenOdd {
		fillBit = FLAGS_FILL_BIT
	}
	return Style{
		flagsAndMiterLimit: fillBit,
		lineWidth:          0,
	}
}

func NewStyleFromStroke(stroke Stroke) Style {
	style := FLAGS_STYLE_BIT
	var join, startCap, endCap uint32
	switch stroke.join {
	case Bevel:
		join = FLAGS_JOIN_BITS_BEVEL
	case Miter:
		join = FLAGS_JOIN_BITS_MITER
	case RoundJoin:
		join = FLAGS_JOIN_BITS_ROUND
	}
	switch stroke.startCap {
	case Butt:
		startCap = FLAGS_START_CAP_BITS_BUTT
	case Square:
		startCap = FLAGS_START_CAP_BITS_SQUARE
	case Round:
		startCap = FLAGS_START_CAP_BITS_ROUND
	}
	switch stroke.endCap {
	case Butt:
		endCap = FLAGS_END_CAP_BITS_BUTT
	case Square:
		endCap = FLAGS_END_CAP_BITS_SQUARE
	case Round:
		endCap = FLAGS_END_CAP_BITS_ROUND
	}
	miterLimit := f32ToF16(float32(stroke.miterLimit))
	return Style{
		flagsAndMiterLimit: style | join | startCap | endCap | miterLimit,
		lineWidth:          float32(stroke.width),
	}
}

func f32ToF16(f float32) uint32 {
	// This is a placeholder implementation. You need to implement the actual conversion.
	return uint32(math.Float32bits(f) >> 16)
}

type Stroke struct {
	width      float64
	join       Join
	startCap   Cap
	endCap     Cap
	miterLimit float64
}

func NewStroke(width float64) Stroke {
	return Stroke{width: width}
}

func (s Stroke) WithJoin(join Join) Stroke {
	s.join = join
	return s
}

func (s Stroke) WithStartCap(cap Cap) Stroke {
	s.startCap = cap
	return s
}

func (s Stroke) WithEndCap(cap Cap) Stroke {
	s.endCap = cap
	return s
}

func (s Stroke) WithMiterLimit(limit float64) Stroke {
	s.miterLimit = limit
	return s
}

type PathTag struct {
	value uint8
}

const (
	LINE_TO_F32     PathTag = 0x9
	QUAD_TO_F32     PathTag = 0xa
	CUBIC_TO_F32    PathTag = 0xb
	LINE_TO_I16     PathTag = 0x1
	QUAD_TO_I16     PathTag = 0x2
	CUBIC_TO_I16    PathTag = 0x3
	TRANSFORM       PathTag = 0x20
	PATH            PathTag = 0x10
	STYLE           PathTag = 0x40
	SUBPATH_END_BIT uint8   = 0x4
	F32_BIT         uint8   = 0x8
	SEGMENT_MASK    uint8   = 0x3
)

func (pt PathTag) IsPathSegment() bool {
	return pt.PathSegmentType().value != 0
}

func (pt PathTag) IsF32() bool {
	return pt.value&F32_BIT != 0
}

func (pt PathTag) IsSubpathEnd() bool {
	return pt.value&SUBPATH_END_BIT != 0
}

func (pt *PathTag) SetSubpathEnd() {
	pt.value |= SUBPATH_END_BIT
}

func (pt PathTag) PathSegmentType() PathSegmentType {
	return PathSegmentType{pt.value & SEGMENT_MASK}
}

type PathSegmentType struct {
	value uint8
}

const (
	LINE_TO  PathSegmentType = 0x1
	QUAD_TO  PathSegmentType = 0x2
	CUBIC_TO PathSegmentType = 0x3
)

type PathMonoid struct {
	transIx       uint32
	pathsegIx     uint32
	pathsegOffset uint32
	styleIx       uint32
	pathIx        uint32
}

func NewPathMonoid(tagWord uint32) PathMonoid {
	var c PathMonoid
	pointCount := tagWord & 0x3030303
	c.pathsegIx = countOnes((pointCount * 7) & 0x4040404)
	c.transIx = countOnes(tagWord & (TRANSFORM.value * 0x1010101))
	nPoints := pointCount + ((tagWord >> 2) & 0x1010101)
	a := nPoints + (nPoints & (((tagWord >> 3) & 0x1010101) * 15))
	a += a >> 8
	a += a >> 16
	c.pathsegOffset = a & 0xff
	c.pathIx = countOnes(tagWord & (PATH.value * 0x1010101))
	styleSize := uint32(unsafe.Sizeof(Style{}) / unsafe.Sizeof(uint32(0)))
	c.styleIx = countOnes(tagWord&(STYLE.value*0x1010101)) * styleSize
	return c
}

func (c PathMonoid) Combine(other PathMonoid) PathMonoid {
	return PathMonoid{
		transIx:       c.transIx + other.transIx,
		pathsegIx:     c.pathsegIx + other.pathsegIx,
		pathsegOffset: c.pathsegOffset + other.pathsegOffset,
		styleIx:       c.styleIx + other.styleIx,
		pathIx:        c.pathIx + other.pathIx,
	}
}

func countOnes(x uint32) uint32 {
	count := uint32(0)
	for x != 0 {
		x &= x - 1
		count++
	}
	return count
}

type PathEncoder struct {
	tags                 *[]PathTag
	data                 *[]byte
	nSegments            *uint32
	nPaths               *uint32
	firstPoint           [2]float32
	firstStartTangentEnd [2]float32
	state                PathState
	nEncodedSegments     uint32
	isFill               bool
}

type PathState int

const (
	Start PathState = iota
	MoveTo
	NonemptySubpath
)

func NewPathEncoder(tags *[]PathTag, data *[]byte, nSegments, nPaths *uint32, isFill bool) *PathEncoder {
	return &PathEncoder{
		tags:                 tags,
		data:                 data,
		nSegments:            nSegments,
		nPaths:               nPaths,
		firstPoint:           [2]float32{0, 0},
		firstStartTangentEnd: [2]float32{0, 0},
		state:                Start,
		nEncodedSegments:     0,
		isFill:               isFill,
	}
}

func (pe *PathEncoder) MoveTo(x, y float32) {
	if pe.isFill {
		pe.Close()
	}
	buf := [2]float32{x, y}
	bytes := (*[8]byte)(unsafe.Pointer(&buf))[:]
	if pe.state == MoveTo {
		newLen := len(*pe.data) - 8
		*pe.data = (*pe.data)[:newLen]
	} else if pe.state == NonemptySubpath {
		if !pe.isFill {
			pe.insertStrokeCapMarkerSegment(false)
		}
		if len(*pe.tags) > 0 {
			(*pe.tags)[len(*pe.tags)-1].SetSubpathEnd()
		}
	}
	pe.firstPoint = buf
	*pe.data = append(*pe.data, bytes...)
	pe.state = MoveTo
}

func (pe *PathEncoder) LineTo(x, y float32) {
	if pe.state == Start {
		if pe.nEncodedSegments == 0 {
			pe.MoveTo(x, y)
			return
		}
		pe.MoveTo(pe.firstPoint[0], pe.firstPoint[1])
	}
	if pe.state == MoveTo {
		if pt := pe.startTangentForLine([2]float32{x, y}); pt != nil {
			pe.firstStartTangentEnd = *pt
		} else {
			return
		}
	}
	if pe.isZeroLengthSegment([2]float32{x, y}, nil, nil) {
		return
	}
	buf := [2]float32{x, y}
	bytes := (*[8]byte)(unsafe.Pointer(&buf))[:]
	*pe.data = append(*pe.data, bytes...)
	*pe.tags = append(*pe.tags, LINE_TO_F32)
	pe.state = NonemptySubpath
	pe.nEncodedSegments++
}

func (pe *PathEncoder) QuadTo(x1, y1, x2, y2 float32) {
	if pe.state == Start {
		if pe.nEncodedSegments == 0 {
			pe.MoveTo(x2, y2)
			return
		}
		pe.MoveTo(pe.firstPoint[0], pe.firstPoint[1])
	}
	if pe.state == MoveTo {
		if pt := pe.startTangentForQuad([2]float32{x1, y1}, [2]float32{x2, y2}); pt != nil {
			pe.firstStartTangentEnd = *pt
		} else {
			return
		}
	}
	if pe.isZeroLengthSegment([2]float32{x1, y1}, &[2]float32{x2, y2}, nil) {
		return
	}
	buf := [4]float32{x1, y1, x2, y2}
	bytes := (*[16]byte)(unsafe.Pointer(&buf))[:]
	*pe.data = append(*pe.data, bytes...)
	*pe.tags = append(*pe.tags, QUAD_TO_F32)
	pe.state = NonemptySubpath
	pe.nEncodedSegments++
}

func (pe *PathEncoder) CubicTo(x1, y1, x2, y2, x3, y3 float32) {
	if pe.state == Start {
		if pe.nEncodedSegments == 0 {
			pe.MoveTo(x3, y3)
			return
		}
		pe.MoveTo(pe.firstPoint[0], pe.firstPoint[1])
	}
	if pe.state == MoveTo {
		if pt := pe.startTangentForCurve([2]float32{x1, y1}, [2]float32{x2, y2}, [2]float32{x3, y3}); pt != nil {
			pe.firstStartTangentEnd = *pt
		} else {
			return
		}
	}
	if pe.isZeroLengthSegment([2]float32{x1, y1}, &[2]float32{x2, y2}, &[2]float32{x3, y3}) {
		return
	}
	buf := [6]float32{x1, y1, x2, y2, x3, y3}
	bytes := (*[24]byte)(unsafe.Pointer(&buf))[:]
	*pe.data = append(*pe.data, bytes...)
	*pe.tags = append(*pe.tags, CUBIC_TO_F32)
	pe.state = NonemptySubpath
	pe.nEncodedSegments++
}

func (pe *PathEncoder) EmptyPath() {
	coords := [4]float32{0, 0, 0, 0}
	bytes := (*[16]byte)(unsafe.Pointer(&coords))[:]
	*pe.data = append(*pe.data, bytes...)
	*pe.tags = append(*pe.tags, LINE_TO_F32)
	pe.nEncodedSegments++
}

func (pe *PathEncoder) Close() {
	switch pe.state {
	case Start:
		return
	case MoveTo:
		newLen := len(*pe.data) - 8
		*pe.data = (*pe.data)[:newLen]
		pe.state = Start
		return
	case NonemptySubpath:
	}
	lenData := len(*pe.data)
	if lenData < 8 {
		return
	}
	firstBytes := (*[8]byte)(unsafe.Pointer(&pe.firstPoint))[:]
	if string((*pe.data)[lenData-8:]) != string(firstBytes) {
		*pe.data = append(*pe.data, firstBytes...)
		*pe.tags = append(*pe.tags, LINE_TO_F32)
		pe.nEncodedSegments++
	}
	if !pe.isFill {
		pe.insertStrokeCapMarkerSegment(true)
	}
	if len(*pe.tags) > 0 {
		(*pe.tags)[len(*pe.tags)-1].SetSubpathEnd()
	}
	pe.state = Start
}

func (pe *PathEncoder) Shape(shape Shape) {
	pe.PathElements(shape.PathElements(0.1))
}

func (pe *PathEncoder) PathElements(path []PathEl) {
	for _, el := range path {
		switch el.Type {
		case MoveTo:
			pe.MoveTo(el.Points[0].X, el.Points[0].Y)
		case LineTo:
			pe.LineTo(el.Points[0].X, el.Points[0].Y)
		case QuadTo:
			pe.QuadTo(el.Points[0].X, el.Points[0].Y, el.Points[1].X, el.Points[1].Y)
		case CurveTo:
			pe.CubicTo(el.Points[0].X, el.Points[0].Y, el.Points[1].X, el.Points[1].Y, el.Points[2].X, el.Points[2].Y)
		case ClosePath:
			pe.Close()
		}
	}
}

func (pe *PathEncoder) Finish(insertPathMarker bool) uint32 {
	if pe.isFill {
		pe.Close()
	}
	if pe.state == MoveTo {
		newLen := len(*pe.data) - 8
		*pe.data = (*pe.data)[:newLen]
	}
	if pe.nEncodedSegments != 0 {
		if !pe.isFill && pe.state == NonemptySubpath {
			pe.insertStrokeCapMarkerSegment(false)
		}
		if len(*pe.tags) > 0 {
			(*pe.tags)[len(*pe.tags)-1].SetSubpathEnd()
		}
		*pe.nSegments += pe.nEncodedSegments
		if insertPathMarker {
			*pe.tags = append(*pe.tags, PATH)
			*pe.nPaths++
		}
	}
	return pe.nEncodedSegments
}

func (pe *PathEncoder) insertStrokeCapMarkerSegment(isClosed bool) {
	if isClosed {
		pe.LineTo(pe.firstStartTangentEnd[0], pe.firstStartTangentEnd[1])
	} else {
		pe.QuadTo(pe.firstPoint[0], pe.firstPoint[1], pe.firstStartTangentEnd[0], pe.firstStartTangentEnd[1])
	}
}

func (pe *PathEncoder) lastPoint() (float32, float32) {
	lenData := len(*pe.data)
	if lenData < 8 {
		return 0, 0
	}
	return *(*float32)(unsafe.Pointer(&(*pe.data)[lenData-8])), *(*float32)(unsafe.Pointer(&(*pe.data)[lenData-4]))
}

func (pe *PathEncoder) isZeroLengthSegment(p1 [2]float32, p2, p3 *[2]float32) bool {
	p0 := [2]float32{pe.lastPoint()}
	if p2 == nil {
		p2 = &p1
	}
	if p3 == nil {
		p3 = &p1
	}
	xMin := math.Min(math.Min(math.Min(float64(p0[0]), float64(p1[0])), float64((*p2)[0])), float64((*p3)[0]))
	xMax := math.Max(math.Max(math.Max(float64(p0[0]), float64(p1[0])), float64((*p2)[0])), float64((*p3)[0]))
	yMin := math.Min(math.Min(math.Min(float64(p0[1]), float64(p1[1])), float64((*p2)[1])), float64((*p3)[1]))
	yMax := math.Max(math.Max(math.Max(float64(p0[1]), float64(p1[1])), float64((*p2)[1])), float64((*p3)[1]))
	return !(xMax-xMin > EPSILON || yMax-yMin > EPSILON)
}

const EPSILON = 1e-12

func (pe *PathEncoder) startTangentForCurve(p1, p2, p3 [2]float32) *[2]float32 {
	p0 := pe.firstPoint
	var pt *[2]float32
	if math.Abs(float64(p1[0]-p0[0])) > EPSILON || math.Abs(float64(p1[1]-p0[1])) > EPSILON {
		pt = &p1
	} else if math.Abs(float64(p2[0]-p0[0])) > EPSILON || math.Abs(float64(p2[1]-p0[1])) > EPSILON {
		pt = &p2
	} else if math.Abs(float64(p3[0]-p0[0])) > EPSILON || math.Abs(float64(p3[1]-p0[1])) > EPSILON {
		pt = &p3
	} else {
		return nil
	}
	return pt
}

func (pe *PathEncoder) startTangentForLine(p1 [2]float32) *[2]float32 {
	p0 := pe.firstPoint
	var pt *[2]float32
	if math.Abs(float64(p1[0]-p0[0])) > EPSILON || math.Abs(float64(p1[1]-p0[1])) > EPSILON {
		pt = &[2]float32{
			p0[0] + 1.0/3.0*(p1[0]-p0[0]),
			p0[1] + 1.0/3.0*(p1[1]-p0[1]),
		}
	} else {
		return nil
	}
	return pt
}

func (pe *PathEncoder) startTangentForQuad(p1, p2 [2]float32) *[2]float32 {
	p0 := pe.firstPoint
	var pt *[2]float32
	if math.Abs(float64(p1[0]-p0[0])) > EPSILON || math.Abs(float64(p1[1]-p0[1])) > EPSILON {
		pt = &[2]float32{
			p1[0] + 1.0/3.0*(p0[0]-p1[0]),
			p1[1] + 1.0/3.0*(p0[1]-p1[1]),
		}
	} else if math.Abs(float64(p2[0]-p0[0])) > EPSILON || math.Abs(float64(p2[1]-p0[1])) > EPSILON {
		pt = &[2]float32{
			p1[0] + 1.0/3.0*(p2[0]-p1[0]),
			p1[1] + 1.0/3.0*(p2[1]-p1[1]),
		}
	} else {
		return nil
	}
	return pt
}
