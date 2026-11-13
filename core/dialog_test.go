// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

func TestDialogMessage(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(300))
	})
	b.AssertRender(t, "dialog/message", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		MessageDialog(b, "Something happened", "Message")
	})
}

// TestDialogMinSize tests that the [Stage.MinSize] of a dialog sets its
// minimum initial size, in dp, larger than its content.
func TestDialogMinSize(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1000), units.Dp(800)) })
	b.runAndShowNewWindow()
	rw := b.Scene.RenderWindow()
	sizeOf := func(minSize math32.Vector2) image.Point {
		d := NewBody()
		NewText(d).SetText("Small")
		b.AsyncLock()
		st := d.NewDialog(b).SetMinSize(minSize).Run()
		b.AsyncUnlock()
		b.waitNoEvents(rw)
		b.AsyncLock()
		defer b.AsyncUnlock()
		sz := st.Scene.SceneGeom.Size
		st.ClosePopup()
		d.Close()
		return sz
	}
	dp := b.Scene.Styles.UnitContext.Dp(1)
	small := sizeOf(math32.Vector2{})
	assert.Less(t, float32(small.X), 700*dp, "the default")
	big := sizeOf(math32.Vec2(700, 500))
	assert.GreaterOrEqual(t, float32(big.X), 700*dp)
	assert.GreaterOrEqual(t, float32(big.Y), 500*dp)
}
