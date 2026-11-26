// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"image"
	"testing"

	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

// TestMouseClone tests that an event made from a mouse event, as a
// context menu from a right mouse up, is a mouse event, with its
// position and button.
func TestMouseClone(t *testing.T) {
	m := NewMouse(MouseUp, Right, image.Pt(50, 60), 0)
	m.Init()
	m.SetHandled()
	e := m.NewFromClone(ContextMenu)
	cm, ok := e.(*Mouse)
	assert.True(t, ok, "a mouse event")
	assert.True(t, e.HasPos())
	assert.Equal(t, image.Pt(50, 60), e.WindowPos())
	assert.Equal(t, ContextMenu, e.Type())
	assert.False(t, e.IsHandled())
	assert.Equal(t, Right, cm.Button)
	assert.Equal(t, MouseUp, m.Type(), "the original is unchanged")
	assert.True(t, m.IsHandled())

	s := NewScroll(image.Pt(1, 2), math32.Vec2(0, 3), 0)
	s.Init()
	sc, ok := s.NewFromClone(Scroll).(*MouseScroll)
	assert.True(t, ok)
	assert.Equal(t, float32(3), sc.Delta.Y)
}
