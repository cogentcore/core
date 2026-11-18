// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"maps"
	"path/filepath"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/sharedfile"
)

// FilePickerSettings are the settings of the [FilePicker] that it keeps
// as the user changes them, such as the widths of its columns, in
// filepicker_settings.json in the Cogent Core data directory, which is
// shared by all apps (see [sharedfile]). It is read again when a file
// picker is opened, if another app has changed it. New settings are
// added as new fields, which are ignored by older versions, and are the
// zero value when read from an older file, so the zero value of each
// field must be its default.
type FilePickerSettings struct {

	// ColumnWidths are the widths of the columns of the table of files,
	// in Dp, by field name, for the columns whose widths the user has set
	// (see [Table.ColumnWidths]); the others have automatic widths.
	ColumnWidths map[string]float32 `json:",omitempty"`
}

// filePickerSettings is the file of the [FilePickerSettings], protected by
// a mutex, as file pickers can be in different windows.
var filePickerSettings struct {
	mu   sync.Mutex
	file sharedfile.File[FilePickerSettings]
}

// filePickerSettingsPath returns the path of the [FilePickerSettings] file.
// It is that of the current app, as it changes in tests.
func filePickerSettingsPath() string {
	return filepath.Join(TheApp.CogentCoreDataDir(), "filepicker_settings.json")
}

// loadFilePickerSettings returns the [FilePickerSettings], read again
// from their file if it has changed since it was last read.
func loadFilePickerSettings() FilePickerSettings {
	fs := &filePickerSettings
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.file.Path = filePickerSettingsPath()
	errors.Log(fs.file.Reload())
	s := fs.file.Data
	s.ColumnWidths = maps.Clone(s.ColumnWidths) // not shared with the file
	return s
}

// updateFilePickerSettings changes the [FilePickerSettings] with the given
// function and saves them, with the changes of other apps (see
// [sharedfile.File.Update]).
func updateFilePickerSettings(change func(s *FilePickerSettings)) error {
	fs := &filePickerSettings
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.file.Path = filePickerSettingsPath()
	return fs.file.Update(change)
}
