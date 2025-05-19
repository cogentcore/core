// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
)

type benchTableStruct struct {
	Icon  icons.Icon
	Age   int
	Score float32
	Name  string
	File  Filename
}

// Note: MUST remove "go" in renderwindow.go call to w.renderAsync!
func BenchmarkTable(bm *testing.B) {
	b := NewBody()
	table := make([]benchTableStruct, 50)
	NewTable(b).SetSlice(&table)
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1280), units.Dp(720))
	})
	b.AssertRender(bm, "table/benchmark", func() {
		w := b.Scene.RenderWindow()
		for range bm.N {
			b.Scene.NeedsRender()
			w.renderWindow() // Note: MUST remove "go" in renderwindow.go call to w.renderAsync!
		}
	})
	return // comment to do profile too
	b.AssertRender(bm, "table/profile", func() {
		w := b.Scene.RenderWindow()
		// startCPUMemoryProfile()
		startTargetedProfile()
		for range 200 {
			b.Scene.NeedsRender()
			w.renderWindow() // Note: MUST remove "go" in renderwindow.go call to w.renderAsync!
		}
		endTargetedProfile()
		// endCPUMemoryProfile()
	})
}

// Note: MUST remove "go" in renderwindow.go call to w.renderAsync!
func BenchmarkForm(bm *testing.B) {
	b := NewBody()
	s := styles.NewStyle()
	s.SetState(true, states.Active)
	s.SetAbilities(true, abilities.Checkable)
	NewForm(b).SetStruct(s)
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1280), units.Dp(720))
	})
	b.AssertRender(bm, "form/benchmark", func() {
		w := b.Scene.RenderWindow()
		for range bm.N {
			b.Scene.NeedsRender()
			w.renderWindow() // Note: MUST remove "go" in renderwindow.go call to w.renderAsync!
		}
	})
	return // comment to do profile too
	b.AssertRender(bm, "form/profile", func() {
		w := b.Scene.RenderWindow()
		// startCPUMemoryProfile()
		startTargetedProfile()
		for range 200 {
			b.Scene.NeedsRender()
			w.renderWindow() // Note: MUST remove "go" in renderwindow.go call to w.renderAsync!
		}
		endTargetedProfile()
		// endCPUMemoryProfile()
	})
}
