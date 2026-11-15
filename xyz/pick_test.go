// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz_test

import (
	"image"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	. "cogentcore.org/core/xyz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPickLines tests finding lines under a point, which must be near the
// line, not just in its bounding box, and that invisible nodes are skipped.
func TestPickLines(t *testing.T) {
	sc := newRenderScene(t)
	var diag, vert, hidden *Solid
	tree.AddChild(sc, func(n *Solid) {
		diag = n
		n.SetName("diag")
		InitLine(n, math32.Vec3(-2, -1.5, 0), math32.Vec3(2, 1.5, 0), 0.05, colors.Green)
	})
	tree.AddChild(sc, func(n *Solid) {
		vert = n
		n.SetName("vert")
		InitLine(n, math32.Vec3(-3, -1, 0), math32.Vec3(-3, 1, 0), 0.05, colors.Green)
	})
	box := NewBox(sc, "box", 1, 1, 1)
	tree.AddChild(sc, func(n *Group) {
		n.SetName("hidden-group")
		n.Invisible = true
		tree.AddChild(n, func(s *Solid) {
			hidden = s
			s.SetMesh(box).SetColor(colors.Red)
		})
	})
	render(t, sc)
	require.False(t, diag.SceneBBox.Empty())

	bb := diag.SceneBBox
	ctr := bb.Min.Add(bb.Max).Div(2)
	assert.Contains(t, NodesUnderPoint(sc, ctr), Node(diag), "on the line")
	// the line goes up to the right on the screen, so the top left corner
	// of its box is far from it, and the top right is on it
	assert.NotContains(t, NodesUnderPoint(sc, image.Pt(bb.Min.X+3, bb.Min.Y+3)), Node(diag), "far from the line")
	assert.Contains(t, NodesUnderPoint(sc, image.Pt(bb.Max.X-3, bb.Min.Y+3)), Node(diag), "near its end")

	vb := vert.SceneBBox
	assert.Contains(t, NodesUnderPoint(sc, vb.Min.Add(vb.Max).Div(2)), Node(vert), "a vertical line")

	assert.NotContains(t, NodesUnderPoint(sc, image.Pt(80, 60)), Node(hidden), "invisible")
}
