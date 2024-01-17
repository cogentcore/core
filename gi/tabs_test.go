// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"

	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

func TestTabs(t *testing.T) {
	configTab := func(tb *Frame) {
		NewLabel(tb).SetType(LabelHeadlineLarge).SetText(tb.Name())
		NewLabel(tb).SetText(testStrings[len(testStrings)-1])
		NewButton(tb).SetText(tb.Name()).SetIcon(icons.Send)
	}
	for _, typ := range TabTypesValues() {
		sc := NewScene()
		sc.Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(800))
		})
		ts := NewTabs(sc).SetType(typ)
		configTab(ts.NewTab("Search", icons.Search))
		configTab(ts.NewTab("Discover", icons.Explore))
		configTab(ts.NewTab("History", icons.History))
		sc.AssertRender(t, testName("tabs", typ))
	}
}
