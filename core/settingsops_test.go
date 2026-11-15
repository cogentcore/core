// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSettingsData are settings for the tests, in a temporary file.
type testSettingsData struct {
	SettingsBase
	Count   int    `default:"3"`
	Name2   string `default:"none"`
	applied int
}

func (ts *testSettingsData) Apply() { ts.applied++ }

// newTestSettings returns test settings in a file of the given extension
// in a temporary directory.
func newTestSettings(t *testing.T, ext string) *testSettingsData {
	return &testSettingsData{Name: "Test", File: filepath.Join(t.TempDir(), "test"+ext)}
}

// TestSettingsSaveLoadReset tests saving, loading, and resetting settings,
// in TOML and JSON, where only the non-default values are saved.
func TestSettingsSaveLoadReset(t *testing.T) {
	skipOnWeb(t, "it saves the settings in files of a temporary directory")
	for _, ext := range []string{".toml", ".json"} {
		ts := newTestSettings(t, ext)
		require.NoError(t, loadSettings(ts), "not saved yet is fine")
		assert.Equal(t, 3, ts.Count, "the default")
		assert.Equal(t, 1, ts.applied)
		assert.Equal(t, "Test", ts.Label())
		assert.Equal(t, ts.File, ts.Filename())

		ts.Count = 7
		require.NoError(t, SaveSettings(ts))
		data, err := os.ReadFile(ts.File)
		require.NoError(t, err)
		assert.Contains(t, string(data), "7")
		assert.NotContains(t, string(data), "none", "default values are not saved")

		ld := newTestSettings(t, ext)
		ld.File = ts.File
		require.NoError(t, loadSettings(ld))
		assert.Equal(t, 7, ld.Count, ext)

		require.NoError(t, resetSettings(ld))
		assert.Equal(t, 3, ld.Count, "reset to the default")
		_, err = os.Stat(ld.File)
		assert.True(t, os.IsNotExist(err), "the file is removed")
	}
	bad := newTestSettings(t, ".toml")
	require.NoError(t, os.WriteFile(bad.File, []byte("Count = ["), 0o644))
	assert.Error(t, loadSettings(bad))
	assert.Equal(t, 1, bad.applied, "applied even with an error")
}

// TestUpdateSettings tests applying and saving settings in a window, and
// adding app settings.
func TestUpdateSettings(t *testing.T) {
	ts := newTestSettings(t, ".toml")
	b := NewBody()
	tx := NewText(b).SetText("settings")
	RenderImage(b, Locked(b, func() {
		ts.Count = 9
		UpdateSettings(tx, ts)
		assert.Equal(t, 1, ts.applied)
		_, err := os.Stat(ts.File)
		assert.NoError(t, err, "saved")
		restyleAll()
	}))

	all := slices.Clone(AllSettings)
	defer func() { AllSettings = all }()
	AddAppSettings(ts)
	assert.Same(t, ts, AllSettings[1], "after the appearance settings")
	assert.Len(t, AllSettings, len(all)+1)
}

// TestSettingsHelpers tests the smaller helpers of the settings.
func TestSettingsHelpers(t *testing.T) {
	force := ForceAppColor
	defer func() { ForceAppColor = force }()
	ForceAppColor = true
	assert.False(t, AppearanceSettings.ShouldDisplay("Color"))
	assert.True(t, AppearanceSettings.ShouldDisplay("Zoom"))
	ForceAppColor = false
	assert.True(t, AppearanceSettings.ShouldDisplay("Color"))

	var fp favoritePaths
	fp.setToDefaults()
	i, ok := fp.findPath("~/Documents")
	assert.True(t, ok)
	assert.Equal(t, "Documents", fp[i].Label())
	_, ok = fp.findPath("/nowhere")
	assert.False(t, ok)

	var db DebugSettingsData
	db.Defaults()
	db.Apply()
	var sb SettingsBase
	sb.Defaults()
	sb.Apply()
	sb.MakeToolbar(nil)
}
