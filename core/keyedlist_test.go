// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"maps"
	"testing"

	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
)

func TestKeyedList(t *testing.T) {
	b := NewBody()
	NewKeyedList(b).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5})
	b.AssertRender(t, "keyed-list/basic")
}

func TestKeyedListInline(t *testing.T) {
	b := NewBody()
	NewKeyedList(b).SetInline(true).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5})
	b.AssertRender(t, "keyed-list/inline")
}

func TestKeyedListReadOnly(t *testing.T) {
	b := NewBody()
	NewKeyedList(b).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5}).SetReadOnly(true)
	b.AssertRender(t, "keyed-list/read-only")
}

func TestKeyedListAny(t *testing.T) {
	b := NewBody()
	NewKeyedList(b).SetMap(&map[string]any{"Go": 1, "C++": "C-like", "Python": true})
	b.AssertRender(t, "keyed-list/any")
}

func TestKeyedListChange(t *testing.T) {
	b := NewBody()
	m := map[string]int{"Go": 1, "C++": 3, "Python": 5}

	n := 0
	value := map[string]int{}
	mv := NewKeyedList(b).SetMap(&m)
	mv.OnChange(func(e events.Event) {
		n++
		maps.Copy(value, m)
	})
	b.AssertRender(t, "keyed-list/change", func() {
		// [3] is value of second row, which is "Go" since it is sorted alphabetically
		mv.Child(3).(*Spinner).TrailingIconButton().Send(events.Click)
		assert.Equal(t, 1, n)
		assert.Equal(t, m, value)
		assert.Equal(t, map[string]int{"Go": 2, "C++": 3, "Python": 5}, m)
	})
}

func TestKeyedListChangeKey(t *testing.T) {
	b := NewBody()
	m := map[string]int{"Go": 1, "C++": 3, "Python": 5}

	n := 0
	value := map[string]int{}
	mv := NewKeyedList(b).SetMap(&m)
	mv.OnChange(func(e events.Event) {
		n++
		maps.Copy(value, m)
	})
	b.AssertRender(t, "keyed-list/change-key", func() {
		// [4] is key of third row, which is "Python" since it is sorted alphabetically
		mv.Child(4).(*TextField).SetText("JavaScript").SendChange()
		assert.Equal(t, 1, n)
		assert.Equal(t, m, value)
		assert.Equal(t, map[string]int{"Go": 1, "C++": 3, "JavaScript": 5}, m)
	})
}
