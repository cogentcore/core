// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"testing"

	"cogentcore.org/core/events/key"
	"github.com/stretchr/testify/assert"
)

func TestDropModFor(t *testing.T) {
	mods := func(ms ...key.Modifiers) key.Modifiers {
		var m key.Modifiers
		for _, f := range ms {
			m.SetFlag(true, f)
		}
		return m
	}
	for _, c := range []struct {
		mods    key.Modifiers
		copyDef DropMods
		moveDef DropMods
	}{
		{0, DropCopy, DropMove},
		{mods(key.Shift), DropMove, DropCopy},
		{mods(key.Meta), DropMove, DropCopy},
		{mods(key.Control), DropCopy, DropCopy},
		{mods(key.Alt), DropLink, DropLink},
	} {
		assert.Equal(t, c.copyDef, DropModFor(c.mods, false), c.mods.String())
		assert.Equal(t, c.moveDef, DropModFor(c.mods, true), c.mods.String())
	}
}
