// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz_test

import (
	"image"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	. "cogentcore.org/core/xyz"
	"github.com/stretchr/testify/assert"
)

// newNavScene returns a scene with a default camera, without a GPU.
func newNavScene() *Scene {
	sc := NewScene()
	sc.Geom.Size = image.Pt(160, 120)
	sc.Camera.Defaults()
	sc.Camera.UpdateMatrix()
	return sc
}

// mods returns the modifiers of the given keys.
func mods(ks ...key.Modifiers) key.Modifiers {
	var m key.Modifiers
	for _, k := range ks {
		m.SetFlag(true, k)
	}
	return m
}

// TestNavKeysAll tests all of the keys that navigate the camera,
// including saving and restoring cameras by number.
func TestNavKeysAll(t *testing.T) {
	sc := newNavScene()
	send := func(rn rune, code key.Codes, ms key.Modifiers) {
		t.Helper()
		e := events.NewKey(events.KeyChord, rn, code, ms)
		sc.NavKeyEvents(e)
		assert.True(t, e.IsHandled(), "%v", e.KeyChord())
	}
	arrows := []key.Codes{key.CodeUpArrow, key.CodeDownArrow, key.CodeLeftArrow, key.CodeRightArrow}
	for _, m := range []key.Modifiers{0, mods(key.Shift), mods(key.Control), mods(key.Alt)} {
		for _, c := range arrows {
			before := sc.Camera.Pose.Pos
			bt := sc.Camera.Target
			send(0, c, m)
			assert.True(t, before != sc.Camera.Pose.Pos || bt != sc.Camera.Target, "%v %v moves", m, c)
		}
	}
	for _, r := range []rune{'+', '=', '-', '_'} {
		before := sc.Camera.Pose.Pos
		send(r, 0, 0)
		assert.NotEqual(t, before, sc.Camera.Pose.Pos, "zoom %c", r)
	}
	for _, r := range []rune{'+', '=', '-', '_'} {
		bt := sc.Camera.Target
		send(r, 0, mods(key.Alt))
		assert.NotEqual(t, bt, sc.Camera.Target, "alt %c pans the target", r)
	}

	// a number saves the camera the first time, and restores it after
	saved := sc.Camera.Pose.Pos
	send('3', key.Code3, 0)
	send(0, key.CodeUpArrow, 0)
	assert.NotEqual(t, saved, sc.Camera.Pose.Pos)
	send('3', key.Code3, 0)
	assert.Equal(t, saved, sc.Camera.Pose.Pos, "restored")
	send(0, key.CodeUpArrow, 0)
	send('3', key.Code3, mods(key.Control)) // saves it again
	moved := sc.Camera.Pose.Pos
	send(0, key.CodeDownArrow, 0)
	send('3', key.Code3, 0)
	assert.Equal(t, moved, sc.Camera.Pose.Pos)

	// space and escape go to the default camera (saved as "default")
	send(' ', key.CodeSpacebar, 0)
	def := sc.Camera.Pose.Pos
	sc.SaveCamera("default")
	send(0, key.CodeUpArrow, 0)
	send(0, key.CodeEscape, 0)
	assert.Equal(t, def, sc.Camera.Pose.Pos)

	e := events.NewKey(events.KeyChord, 'q', key.CodeQ, 0)
	sc.NavKeyEvents(e)
	assert.False(t, e.IsHandled(), "not a navigation key")
}

// TestSlideScroll tests orbiting and panning the camera by sliding the
// mouse, and zooming by scrolling.
func TestSlideScroll(t *testing.T) {
	sc := newNavScene()
	slide := func(dx, dy int, ms key.Modifiers) {
		e := events.NewMouseMove(events.Left, image.Pt(80+dx, 60+dy), image.Pt(80, 60), ms)
		e.SetLocalOff(image.Point{}) // as the event manager does
		sc.SlideMoveEvent(e)
	}
	for _, m := range []key.Modifiers{0, mods(key.Shift), mods(key.Control), mods(key.Alt)} {
		before := sc.Camera.Pose.Pos
		bt := sc.Camera.Target
		slide(10, 2, m)
		slide(1, -10, m)
		assert.True(t, before != sc.Camera.Pose.Pos || bt != sc.Camera.Target, "%v moves", m)
	}

	dist := sc.Camera.DistanceTo(sc.Camera.Target)
	e := events.NewScroll(image.Pt(80, 60), math32.Vec2(0, 5), 0)
	e.SetLocalOff(image.Point{})
	sc.MouseScrollEvent(e)
	assert.True(t, e.IsHandled())
	assert.NotEqual(t, dist, sc.Camera.DistanceTo(sc.Camera.Target), "zoomed")
	bt := sc.Camera.Target
	sc.MouseScrollEvent(events.NewScroll(image.Pt(80, 60), math32.Vec2(0, 5), mods(key.Alt)))
	assert.NotEqual(t, bt, sc.Camera.Target, "alt moves the target")

	sc.NoNav = true
	e = events.NewScroll(image.Pt(80, 60), math32.Vec2(0, 5), 0)
	sc.MouseScrollEvent(e)
	assert.False(t, e.IsHandled(), "no navigation")
}
