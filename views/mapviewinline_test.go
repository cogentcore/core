// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"maps"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
)

func TestMapViewInline(t *testing.T) {
	b := core.NewBody()
	NewMapViewInline(b).SetMap(&map[string]int{"Go": 1, "C++": 3})
	b.AssertRender(t, "map-view-inline/basic")
}

func TestMapViewInlineReadOnly(t *testing.T) {
	b := core.NewBody()
	NewMapViewInline(b).SetMap(&map[string]int{"Go": 1, "C++": 3}).SetReadOnly(true)
	b.AssertRender(t, "map-view-inline/read-only")
}

func TestMapViewInlineChange(t *testing.T) {
	b := core.NewBody()
	m := map[string]int{"Go": 1, "C++": 3}

	n := 0
	value := map[string]int{}
	mv := NewMapViewInline(b).SetMap(&m)
	mv.OnChange(func(e events.Event) {
		n++
		maps.Copy(value, m)
	})
	b.AssertRender(t, "map-view-inline/change", func() {
		// [3] is value of second row, which is "Go" since it is sorted alphabetically
		mv.Child(3).(*core.Spinner).TrailingIconButton().Send(events.Click)
		assert.Equal(t, 1, n)
		assert.Equal(t, m, value)
		assert.Equal(t, map[string]int{"Go": 2, "C++": 3}, m)
	})
}

func TestMapViewInlineChangeKey(t *testing.T) {
	b := core.NewBody()
	m := map[string]int{"Go": 1, "C++": 3}

	n := 0
	value := map[string]int{}
	mv := NewMapViewInline(b).SetMap(&m)
	mv.OnChange(func(e events.Event) {
		n++
		maps.Copy(value, m)
	})
	b.AssertRender(t, "map-view-inline/change-key", func() {
		// [0] is key of first row, which is "C++" since it is sorted alphabetically
		mv.Child(0).(*core.TextField).SetText("JavaScript").SendChange()
		assert.Equal(t, 1, n)
		assert.Equal(t, m, value)
		assert.Equal(t, map[string]int{"Go": 1, "JavaScript": 3}, m)
	})
}
