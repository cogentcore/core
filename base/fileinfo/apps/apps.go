// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package apps provides access to the apps of the system that open files:
// the apps that can open a file, its default app, opening it with another
// app, and changing the default app for its type of file, using the
// facilities of each platform: NSWorkspace on macOS, and the freedesktop
// (XDG) desktop entries and mimeapps.list files on Linux and other unix
// systems. Other platforms return [errors.ErrUnsupported].
package apps

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
)

// App is an app of the system that can open files.
type App struct {

	// Name is the name of the app shown to the user.
	Name string

	// ID identifies the app to the system: the path of its app bundle
	// on macOS, and the ID of its desktop entry on Linux, as in
	// org.gnome.TextEditor.desktop.
	ID string

	// Default is whether the app is the default app for the type of file.
	Default bool
}

// For returns the apps that can open the given file, with its default app,
// if any, first.
func For(file string) ([]App, error) {
	return forFile(abs(file))
}

// Default returns the default app for the given file, and false if it has none.
func Default(file string) (App, bool, error) {
	as, err := For(file)
	if err != nil || len(as) == 0 || !as[0].Default {
		return App{}, false, err
	}
	return as[0], true, nil
}

// All returns all of the apps of the system that open files, sorted by
// name, as for choosing one that does not declare that it opens a file,
// which can still open it.
func All() ([]App, error) {
	as, err := allApps()
	slices.SortFunc(as, func(a, b App) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return as, err
}

// OpenWith opens the given file with the given app.
func OpenWith(file string, app App) error {
	if app.ID == "" {
		return errors.New("apps: no app to open " + file + " with")
	}
	return openWith(abs(file), app)
}

// SetDefault makes the given app the default app for the type of the given
// file, as for all files with its extension.
func SetDefault(file string, app App) error {
	if app.ID == "" {
		return errors.New("apps: no app to make the default for " + file)
	}
	return setDefault(abs(file), app)
}

// abs returns the absolute path of the given file.
func abs(file string) string {
	if a, err := filepath.Abs(file); err == nil {
		return a
	}
	return file
}
