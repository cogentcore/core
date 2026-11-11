// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInspectorTreeWidth tests that the nodes of the tree of an inspector
// are as wide as its panel, so that long names are not cut short.
func TestInspectorTreeWidth(t *testing.T) {
	inspected := NewBody()
	NewText(inspected).SetName("a-text-with-a-long-name-that-is-not-cut-short-anymore")
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(600)) })
	is := NewInspector(b)
	is.SetRoot(inspected)
	b.RenderImage(func() {}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		var frame *Frame
		var node *Tree
		is.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
			if f, ok := cw.(*Frame); ok && f.Name == "tree-frame" {
				frame = f
			}
			if tr, ok := cw.(*Tree); ok && tr.SyncNode != nil && tr.SyncNode.AsTree().Name == "a-text-with-a-long-name-that-is-not-cut-short-anymore" {
				node = tr
			}
			return tree.Continue
		})
		require.NotNil(t, frame)
		require.NotNil(t, node)
		em := frame.Styles.UnitContext.Dots(units.UnitEm)
		assert.Greater(t, node.Geom.Size.Actual.Total.X, 20*em, "wider than the old limit of 20em")
		assert.InDelta(t, frame.Geom.Size.Actual.Content.X, node.Geom.Size.Actual.Total.X, float64(frame.Geom.Size.Actual.Content.X/4), "about as wide as its panel")
	})
}
