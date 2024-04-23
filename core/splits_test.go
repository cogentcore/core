// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/styles"
)

func TestSplits(t *testing.T) {
	b := NewBody()
	sp := NewSplits(b)
	NewText(sp).SetText("First")
	NewText(sp).SetText("Second")
	b.AssertRender(t, "splits/basic")
}

func TestSplitsMany(t *testing.T) {
	b := NewBody()
	sp := NewSplits(b)
	NewText(sp).SetText("First")
	NewText(sp).SetText("Second")
	NewText(sp).SetText("Third")
	NewText(sp).SetText("Fourth")
	b.AssertRender(t, "splits/many")
}

func TestSplitsSet(t *testing.T) {
	b := NewBody()
	sp := NewSplits(b).SetSplits(0.2, 0.8)
	NewText(sp).SetText("First")
	NewText(sp).SetText("Second")
	b.AssertRender(t, "splits/set")
}

func TestSplitsColumn(t *testing.T) {
	b := NewBody()
	sp := NewSplits(b)
	sp.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	NewText(sp).SetText("First")
	NewText(sp).SetText("Second")
	b.AssertRender(t, "splits/column")
}

func TestSplitsRow(t *testing.T) {
	b := NewBody()
	sp := NewSplits(b)
	sp.Style(func(s *styles.Style) {
		s.Direction = styles.Row
	})
	NewText(sp).SetText("First")
	NewText(sp).SetText("Second")
	b.AssertRender(t, "splits/row")
}

// For https://github.com/cogentcore/core/issues/850
func TestMixedVerticalSplits(t *testing.T) {
	b := NewBody()
	txt := "This is a long sentence that I wrote for the purpose of testing vertical splits behavior"
	NewText(b).SetText(txt)
	sp := NewSplits(b)
	sp.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	NewText(sp).SetText(txt)
	NewText(sp).SetText(txt)
	NewText(b).SetText(txt)
	b.AssertRender(t, "splits/mixed-vertical")
}
