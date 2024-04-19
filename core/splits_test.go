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
	NewLabel(sp).SetText("First")
	NewLabel(sp).SetText("Second")
	b.AssertRender(t, "splits/basic")
}

func TestSplitsMany(t *testing.T) {
	b := NewBody()
	sp := NewSplits(b)
	NewLabel(sp).SetText("First")
	NewLabel(sp).SetText("Second")
	NewLabel(sp).SetText("Third")
	NewLabel(sp).SetText("Fourth")
	b.AssertRender(t, "splits/many")
}

func TestSplitsSet(t *testing.T) {
	b := NewBody()
	sp := NewSplits(b).SetSplits(0.2, 0.8)
	NewLabel(sp).SetText("First")
	NewLabel(sp).SetText("Second")
	b.AssertRender(t, "splits/set")
}

func TestSplitsColumn(t *testing.T) {
	b := NewBody()
	sp := NewSplits(b)
	sp.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	NewLabel(sp).SetText("First")
	NewLabel(sp).SetText("Second")
	b.AssertRender(t, "splits/column")
}

func TestSplitsRow(t *testing.T) {
	b := NewBody()
	sp := NewSplits(b)
	sp.Style(func(s *styles.Style) {
		s.Direction = styles.Row
	})
	NewLabel(sp).SetText("First")
	NewLabel(sp).SetText("Second")
	b.AssertRender(t, "splits/row")
}

// For https://github.com/cogentcore/core/issues/850
func TestMixedVerticalSplits(t *testing.T) {
	b := NewBody()
	txt := "This is a long sentence that I wrote for the purpose of testing vertical splits behavior"
	NewLabel(b).SetText(txt)
	sp := NewSplits(b)
	sp.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	NewLabel(sp).SetText(txt)
	NewLabel(sp).SetText(txt)
	NewLabel(b).SetText(txt)
	b.AssertRender(t, "splits/mixed-vertical")
}
