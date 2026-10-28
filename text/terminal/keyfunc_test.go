// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"testing"

	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
)

// withPlatform runs the given function with the given system platform
// and key map.
func withPlatform(platform string, km keymap.MapName, fun func()) {
	prevPlat, prevMap := key.SystemPlatform, keymap.ActiveMapName
	key.SystemPlatform = platform
	keymap.SetActiveMapName(km)
	defer func() {
		key.SystemPlatform = prevPlat
		keymap.SetActiveMapName(prevMap)
	}()
	fun()
}

func TestKeyFunctionMac(t *testing.T) {
	withPlatform("MacOS", "MacStandard", func() {
		assert.Equal(t, keymap.Copy, keyFunctionOf("Meta+C"))
		assert.Equal(t, keymap.Paste, keyFunctionOf("Meta+V"))
		assert.Equal(t, keymap.Kill, keyFunctionOf("Meta+K"))
		assert.Equal(t, keymap.New, keyFunctionOf("Meta+N"))
		assert.Equal(t, keymap.None, keyFunctionOf("Control+K"), "to the program")
		assert.Equal(t, keymap.None, keyFunctionOf("Control+Shift+K"), "to the program")
		assert.Equal(t, keymap.None, keyFunctionOf("K"))
		assert.Equal(t, key.Chord("Meta+C"), ShortcutChord(keymap.Copy))
		assert.Equal(t, key.Chord("Meta+K"), ShortcutChord(keymap.Kill), "not Control+K")
		assert.Equal(t, key.Chord("Meta+N"), ShortcutChord(keymap.New))
	})
}

func TestKeyFunctionLinux(t *testing.T) {
	withPlatform("Linux", "LinuxStandard", func() {
		assert.Equal(t, keymap.Copy, keyFunctionOf("Control+Shift+C"))
		assert.Equal(t, keymap.Paste, keyFunctionOf("Control+Shift+V"))
		assert.Equal(t, keymap.Kill, keyFunctionOf("Control+Shift+K"))
		assert.Equal(t, keymap.New, keyFunctionOf("Control+Shift+N"))
		assert.Equal(t, keymap.None, keyFunctionOf("Control+C"), "to the program")
		assert.Equal(t, keymap.None, keyFunctionOf("Control+N"), "to the program")
		assert.Equal(t, keymap.None, keyFunctionOf("Control+Shift+UpArrow"), "not a letter")
		assert.Equal(t, key.Chord("Control+Shift+C"), ShortcutChord(keymap.Copy))
		assert.Equal(t, key.Chord("Control+Shift+K"), ShortcutChord(keymap.Kill))
		assert.Equal(t, key.Chord("Control+Shift+N"), ShortcutChord(keymap.New))
	})
}
