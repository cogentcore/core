// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg_test

import (
	"bytes"
	"testing"

	"cogentcore.org/core/math32"
	. "cogentcore.org/core/svg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetNodePosSize tests setting the upper left and the size of nodes,
// as for drawing a new one by dragging: circles and ellipses, whose
// position is their center, keep their upper left where it is set.
func TestSetNodePosSize(t *testing.T) {
	sv := NewSVG(math32.Vec2(100, 100))
	c := NewCircle(sv.Root)
	c.SetNodePos(math32.Vec2(10, 20))
	c.SetNodeSize(math32.Vec2(8, 12))
	assert.Equal(t, float32(5), c.Radius)
	assert.Equal(t, math32.Vec2(15, 25), c.Pos, "the center")

	e := NewEllipse(sv.Root)
	e.SetNodePos(math32.Vec2(10, 20))
	e.SetNodeSize(math32.Vec2(8, 12))
	assert.Equal(t, math32.Vec2(4, 6), e.Radii)
	assert.Equal(t, math32.Vec2(14, 26), e.Pos, "the center")
	e.SetNodeSize(math32.Vec2(2, 2))
	assert.Equal(t, math32.Vec2(11, 21), e.Pos, "the upper left stays")

	r := NewRect(sv.Root)
	r.SetNodePos(math32.Vec2(1, 2))
	r.SetNodeSize(math32.Vec2(3, 4))
	assert.Equal(t, math32.Vec2(1, 2), r.Pos)
	assert.Equal(t, math32.Vec2(3, 4), r.Size)

	tx := NewText(sv.Root)
	sp := NewText(tx)
	sp.Text = "a"
	tx.SetNodePos(math32.Vec2(5, 6))
	assert.Equal(t, math32.Vec2(5, 6), tx.Pos)
	assert.Equal(t, math32.Vec2(5, 6), sp.Pos, "with no transform")

	// no-ops for the nodes that do not support them
	pl := NewPolyline(sv.Root)
	pl.Points = []math32.Vector2{{0, 0}, {1, 1}}
	pl.SetPos(math32.Vec2(9, 9))
	pl.SetSize(math32.Vec2(9, 9))
	pl.SetNodePos(math32.Vec2(9, 9))
	pl.SetNodeSize(math32.Vec2(9, 9))
	assert.Equal(t, []math32.Vector2{{0, 0}, {1, 1}}, pl.Points)
	NewPath(sv.Root).SetPos(math32.Vec2(1, 1))
}

// TestBitClone tests that a bit copy of a node, as saved before an edit,
// does not share its points or path data with the node, so that the node
// can be restored from it.
func TestBitClone(t *testing.T) {
	sv := NewSVG(math32.Vec2(100, 100))
	pl := NewPolyline(sv.Root)
	pl.Points = []math32.Vector2{{0, 0}, {10, 10}}
	pg := NewPolygon(sv.Root)
	pg.Points = []math32.Vector2{{0, 0}, {10, 0}, {5, 5}}
	for _, n := range []Node{pl, pg} {
		saved := BitCloneNode(n)
		n.ApplyTransform(sv, math32.Translate2D(1, 2))
		var pts, spts []math32.Vector2
		switch x := n.(type) {
		case *Polyline:
			pts, spts = x.Points, saved.(*Polyline).Points
		case *Polygon:
			pts, spts = x.Points, saved.(*Polygon).Points
		}
		assert.NotEqual(t, pts, spts, "the saved points are not moved")
		BitCopyFrom(n, saved)
		switch x := n.(type) {
		case *Polyline:
			assert.Equal(t, math32.Vec2(0, 0), x.Points[0], "restored")
		case *Polygon:
			assert.Equal(t, math32.Vec2(0, 0), x.Points[0], "restored")
		}
	}

	p := NewPath(sv.Root)
	p.SetData("M0 0L10 10")
	saved := BitCloneNode(p).(*Path)
	p.ApplyTransform(sv, math32.Translate2D(1, 2))
	assert.Equal(t, "M0 0L10 10", saved.Data.ToSVG())
}

// TestTextApplyTransform tests transforming text, which moves it, or
// sets its transform when it rotates or scales.
func TestTextApplyTransform(t *testing.T) {
	sv := NewSVG(math32.Vec2(100, 100))
	tx := NewText(sv.Root)
	tx.Text = "a"
	tx.Pos = math32.Vec2(1, 2)
	tx.ApplyTransform(sv, math32.Translate2D(3, 4))
	assert.Equal(t, math32.Vec2(4, 6), tx.Pos, "moved")
	tx.ApplyTransform(sv, math32.Scale2D(2, 2))
	assert.Equal(t, math32.Vec2(4, 6), tx.Pos)
	assert.Equal(t, math32.Scale2D(2, 2), tx.Paint.Transform)
	assert.Equal(t, "scale(2,2)", tx.Properties["transform"])
	NewGroup(sv.Root).AsNodeBase().ApplyTransform(sv, math32.Scale2D(2, 2))
}

// TestXMLEscape tests escaping the text of elements when writing.
func TestXMLEscape(t *testing.T) {
	var b bytes.Buffer
	xe := NewXMLEncoder(&b)
	xe.EscapeString("a\"b'c&d<e>f\tg\nh\ri\x01j�k", true)
	xe.Flush()
	assert.Equal(t, "a&#34;b&#39;c&amp;d&lt;e&gt;f&#x9;g&#xA;h&#xD;i�j�k", b.String())
	b.Reset()
	xe.EscapeString("x\ny", false)
	xe.Flush()
	assert.Equal(t, "x\ny", b.String(), "newlines kept")
}

// TestPreserveAspectRatioErrors tests the errors of parsing the preserved
// aspect ratio of a view box.
func TestPreserveAspectRatioErrors(t *testing.T) {
	var pa ViewBoxPreserveAspectRatio
	require.NoError(t, pa.SetString("xMaxYMin slice"))
	assert.Equal(t, Slice, pa.MeetOrSlice)
	require.NoError(t, pa.SetString("none"))
	require.NoError(t, pa.SetString(""))
	assert.Error(t, pa.SetString("xMid bad"))
	assert.Error(t, pa.SetString("xMidYMid other"))
	assert.Error(t, pa.SetString("xFooYBar"))
}

// TestLengthUnits tests reading lengths in the units of CSS, and in
// percent of the view box.
func TestLengthUnits(t *testing.T) {
	sv := openString(t, `<svg viewBox="0 0 200 100">
<rect id="r1" x="1in" y="2pt" width="1cm" height="10mm"/>
<rect id="r2" x="1pc" y="1em" width="50%" height="50%"/>
<circle id="c3" cx="1px" cy="2" r="10%"/>
</svg>`)
	r1 := sv.FindNamedElement("r1").(*Rect)
	assert.InDelta(t, 96, r1.Pos.X, 1e-3)
	assert.InDelta(t, 8.0/3, r1.Pos.Y, 1e-3)
	assert.InDelta(t, 96/2.54, r1.Size.X, 1e-3)
	assert.InDelta(t, 96/2.54, r1.Size.Y, 1e-3)
	r2 := sv.FindNamedElement("r2").(*Rect)
	assert.InDelta(t, 16, r2.Pos.X, 1e-3)
	assert.InDelta(t, 16, r2.Pos.Y, 1e-3)
	assert.Equal(t, math32.Vec2(100, 50), r2.Size)
	c3 := sv.FindNamedElement("c3").(*Circle)
	assert.InDelta(t, 10*math32.Sqrt((200*200+100*100)/2)/100, c3.Radius, 1e-3, "of the normalized diagonal")
}
