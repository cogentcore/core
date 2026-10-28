// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"testing"

	"cogentcore.org/core/events/key"
	"github.com/stretchr/testify/assert"
)

func TestModifiers(t *testing.T) {
	var mods key.Modifiers
	mods.SetFlag(true, key.Control)
	e := NewKey(KeyChord, 'c', key.CodeC, mods)
	assert.True(t, e.HasAnyModifier(key.Control, key.Shift))
	assert.False(t, e.HasAllModifiers(key.Control, key.Shift))
	assert.True(t, e.HasAllModifiers(key.Control))
	e.Mods.SetFlag(true, key.Shift)
	assert.True(t, e.HasAllModifiers(key.Control, key.Shift))
}
