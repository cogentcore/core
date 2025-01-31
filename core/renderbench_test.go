// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"testing"

	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
)

type benchTableStruct struct {
	Icon  icons.Icon
	Age   int `default:"2"`
	Score float32
	Name  string
	File  Filename
}

func BenchmarkTable(bm *testing.B) {
	b := NewBody()
	table := make([]benchTableStruct, 50)
	NewTable(b).SetSlice(&table)
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1280), units.Dp(720))
	})
	b.AssertRender(bm, "table/benchmark", func() {
		fmt.Println("fun")
		b.AsyncLock()
		startCPUMemoryProfile()
		for range bm.N {
			b.Scene.RenderWidget()
		}
		endCPUMemoryProfile()
		b.AsyncUnlock()
	})
}

func BenchmarkStyleForm(bm *testing.B) {
	b := NewBody()
	s := styles.NewStyle()
	s.SetState(true, states.Active)
	s.SetAbilities(true, abilities.Checkable)
	NewForm(b).SetStruct(s)
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1280), units.Dp(720))
	})
	b.AssertRender(bm, "form/stylebenchmark", func() {
		fmt.Println("fun")
		b.AsyncLock()
		startCPUMemoryProfile()
		for range bm.N {
			b.Scene.RenderWidget()
		}
		endCPUMemoryProfile()
		b.AsyncUnlock()
	})
}
