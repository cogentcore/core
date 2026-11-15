// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAllWidgets tests that iterating over the widgets includes their
// parts, as WidgetWalkDown does, and that breaking out stops it.
func TestAllWidgets(t *testing.T) {
	b := NewBody()
	tr := NewTree(b).SetText("root") // a tree has parts
	NewTree(tr).SetText("child")
	NewText(b).SetText("after")
	RenderImage(b, Locked(b, func() {
		var walked, ranged []Widget
		b.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
			walked = append(walked, cw)
			return true
		})
		for cw, cwb := range b.AllWidgets() {
			assert.Same(t, cw.AsWidget(), cwb)
			ranged = append(ranged, cw)
		}
		assert.Equal(t, walked, ranged, "the same widgets as WidgetWalkDown")
		hasPart := false
		for _, w := range ranged {
			if tr.Parts != nil && w.AsTree().Parent == tr.Parts.This {
				hasPart = true
			}
		}
		assert.True(t, hasPart, "the parts of the tree")

		n := 0
		for cw := range b.AllWidgets() {
			n++
			if cw == Widget(tr) {
				break
			}
		}
		assert.Less(t, n, len(ranged), "stops at the tree")
	}))
}
