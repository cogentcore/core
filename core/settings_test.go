// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSettingsFile opens the given settings file in testdata/settings,
// which was written by the previous pelletier/go-toml/v2 based version
// of tomlx (see testdata/settings/pelletier/main.go), on top of the
// defaults, and checks that saving the settings again writes the same file.
func testSettingsFile(t *testing.T, se Settings, file string) {
	t.Helper()
	filename := filepath.Join("testdata", "settings", file)
	reflectx.SetFromDefaultTags(se)
	se.Defaults()
	if ss, ok := se.(*SystemSettingsData); ok {
		ss.User = User{} // not machine-dependent
	}
	require.NoError(t, tomlx.Open(se, filename))

	want, err := os.ReadFile(filename)
	require.NoError(t, err)
	b, err := tomlx.WriteBytes(reflectx.NonDefaultFields(se))
	require.NoError(t, err)
	assert.Equal(t, string(want), string(b))
}

func TestAppearanceSettingsFile(t *testing.T) {
	as := &AppearanceSettingsData{}
	testSettingsFile(t, as, "appearance.toml")
	assert.Equal(t, ThemeDark, as.Theme)
	assert.Equal(t, color.RGBA{200, 50, 100, 255}, as.Color)
	assert.Equal(t, map[string]ScreenSettings{"Built-in Retina Display": {Zoom: 150}, "DELL U2720Q": {Zoom: 90}}, as.Screens)
	assert.Equal(t, 3, as.InlineLengths.Map)
	assert.Equal(t, 4, as.InlineLengths.Struct)
}

func TestSystemSettingsFile(t *testing.T) {
	ss := &SystemSettingsData{}
	testSettingsFile(t, ss, "system.toml")
	assert.Equal(t, keymap.MapName("MacEmacs"), ss.KeyMap)
	assert.True(t, ss.KeyMaps.Valid)
	assert.Equal(t, keymap.StandardMaps[:2], ss.KeyMaps.Value)
	assert.Equal(t, 8, ss.Editor.TabSize)
	assert.Equal(t, "test@example.com", ss.User.Email)
	assert.Equal(t, "Test User", ss.User.Name)
	assert.Len(t, ss.FavPaths, 6)
	assert.Equal(t, "~/work", ss.FavPaths[5].Path)
}

func TestTimingSettingsFile(t *testing.T) {
	ts := &TimingSettingsData{}
	testSettingsFile(t, ts, "timing.toml")
	assert.Equal(t, 10*time.Second, ts.SnackbarTimeout)
	assert.Equal(t, 400*time.Millisecond, ts.DoubleClickInterval)
	assert.Equal(t, time.Duration(0), ts.CursorBlinkTime)
	assert.Equal(t, 1500*time.Millisecond, ts.LongHoverTime)
	assert.Equal(t, time.Second, ts.ScrollFocusTime)
	assert.Equal(t, float32(2.5), ts.ScrollWheelSpeed)
}
