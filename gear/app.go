// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gear provides the generation of GUIs and interactive CLIs for any existing command line tools.
package gear

import (
	"goki.dev/glop/sentencecase"
)

// App contains all of the data for a parsed command line application.
type App struct {
	// Command is the actual name of the executable for the app (eg: "git")
	Command string
	// Name is the formatted name of the app (eg: "Git")
	Name string
}

// NewApp makes a new [App] object from the given command name.
// It does not parse it; see [App.Parse].
func NewApp(cmd string) *App {
	return &App{
		Command: cmd,
		Name:    sentencecase.Of(cmd),
	}
}
