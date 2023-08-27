// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cmd contains the actual command defintions
// for the commands in the GoKi tool.
package cmd

import "goki.dev/goki/config"

// App is the main app type that handles
// the logic for the GoKi tool
type App config.Config

// TheApp is the singular instance of [App]
var TheApp = &App{}

// Config returns the app as a config object
func (a *App) Config() *config.Config {
	return (*config.Config)(a)
}
