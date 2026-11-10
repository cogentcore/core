// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"strings"
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPreLayout tests that a widget laid out by [Scene.PreLayout] in
// another goroutine (while the scene renders) is laid out as one that is
// not once it is added to its parent, and that its texts are then laid
// out from the caches of the text shaper.
func TestPreLayout(t *testing.T) {
	makeContent := func() *Frame {
		fr := NewFrame()
		fr.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 0)
		})
		for i := range 20 {
			tx := NewText(fr).SetText(strings.Repeat("words that wrap in the width of the frame ", 5+i%3))
			tx.SetName("text" + string(rune('a'+i)))
		}
		row := NewFrame(fr)
		NewButton(row).SetText("button")
		NewTextField(row).SetText("field")
		return fr
	}
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(400), units.Dp(300))
	})
	host := NewFrame(b)
	host.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto)
	})
	// geoms returns the positions and sizes of the widgets in the content,
	// relative to it, in their order
	geoms := func(fr *Frame) []string {
		var gs []string
		origin := fr.Geom.Pos.Total
		fr.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
			p := cwb.Geom.Pos.Total.Sub(origin)
			sz := cwb.Geom.Size.Actual.Total
			gs = append(gs, cwb.Name+" "+p.String()+" "+sz.String())
			return tree.Continue
		})
		return gs
	}
	pre, plain := makeContent(), makeContent()
	var preGeoms []string
	b.RenderImage(func() {
		b.AsyncLock()
		pl := b.Scene.PreLayout(host)
		b.AsyncUnlock()
		require.NotNil(t, pl)
		done := make(chan bool)
		go func() {
			pl(pre)
			done <- true
		}()
		for range 5 { // the scene is laid out meanwhile
			b.AsyncLock()
			b.Scene.LayoutScene()
			b.AsyncUnlock()
		}
		<-done
		assert.Nil(t, pre.Parent, "it is in no scene")
		b.AsyncLock()
		host.AddChild(pre)
		host.Update()
		b.AsyncUnlock()
	}, func() {
		b.AsyncLock()
		preGeoms = geoms(pre)
		pre.Delete()
		host.AddChild(plain)
		host.Update()
		b.AsyncUnlock()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Same(t, b.Scene, pre.Scene, "it was in the scene")
		assert.Equal(t, geoms(plain), preGeoms, "laid out as without PreLayout")
		assert.Greater(t, len(preGeoms), 20)
	})
}
