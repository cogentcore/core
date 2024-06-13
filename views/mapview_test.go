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

func TestKeyValueTable(t *testing.T) {
	b := core.NewBody()
	NewKeyValueTable(b).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5})
	b.AssertRender(t, "key-value-table/basic")
}

func TestKeyValueTableInline(t *testing.T) {
	b := core.NewBody()
	NewKeyValueTable(b).SetInline(true).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5})
	b.AssertRender(t, "key-value-table/inline")
}

func TestKeyValueTableReadOnly(t *testing.T) {
	b := core.NewBody()
	NewKeyValueTable(b).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5}).SetReadOnly(true)
	b.AssertRender(t, "key-value-table/read-only")
}

func TestKeyValueTableAny(t *testing.T) {
	b := core.NewBody()
	NewKeyValueTable(b).SetMap(&map[string]any{"Go": 1, "C++": "C-like", "Python": true})
	b.AssertRender(t, "key-value-table/any")
}

func TestKeyValueTableChange(t *testing.T) {
	b := core.NewBody()
	m := map[string]int{"Go": 1, "C++": 3, "Python": 5}

	n := 0
	value := map[string]int{}
	mv := NewKeyValueTable(b).SetMap(&m)
	mv.OnChange(func(e events.Event) {
		n++
		maps.Copy(value, m)
	})
	b.AssertRender(t, "key-value-table/change", func() {
		// [3] is value of second row, which is "Go" since it is sorted alphabetically
		mv.Child(3).(*core.Spinner).TrailingIconButton().Send(events.Click)
		assert.Equal(t, 1, n)
		assert.Equal(t, m, value)
		assert.Equal(t, map[string]int{"Go": 2, "C++": 3, "Python": 5}, m)
	})
}

func TestKeyValueTableChangeKey(t *testing.T) {
	b := core.NewBody()
	m := map[string]int{"Go": 1, "C++": 3, "Python": 5}

	n := 0
	value := map[string]int{}
	mv := NewKeyValueTable(b).SetMap(&m)
	mv.OnChange(func(e events.Event) {
		n++
		maps.Copy(value, m)
	})
	b.AssertRender(t, "key-value-table/change-key", func() {
		// [4] is key of third row, which is "Python" since it is sorted alphabetically
		mv.Child(4).(*core.TextField).SetText("JavaScript").SendChange()
		assert.Equal(t, 1, n)
		assert.Equal(t, m, value)
		assert.Equal(t, map[string]int{"Go": 1, "C++": 3, "JavaScript": 5}, m)
	})
}
