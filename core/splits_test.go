// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/styles"
)

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
