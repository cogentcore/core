// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"goki.dev/enums/enumgen"
	"goki.dev/enums/enumgen/config"
)

// App is the main app type that handles
// the logic for the enumgen tool
type App config.Config

// TheApp is the singular instance of [App]
var TheApp = &App{}

// Config returns the app as a config object
func (a *App) Config() *config.Config {
	return (*config.Config)(a)
}

// RootCmd is the root command of enumgen
// that generates the enum methods
func (a *App) RootCmd() error {
	return enumgen.Generate(a.Config())
}
