// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package main provides the actual command line
// implementation of the enumgen library.
package main

import (
	"goki.dev/enums/enumgen"
	"goki.dev/grease"
)

// App is the main app type that handles
// the logic for the enumgen tool
type App struct{}

// Cmd is the main command of enumgen
// that generates the enum methods
func (a *App) Cmd(cfg *enumgen.Config) error {
	return enumgen.Generate(cfg)
}

func main() {
	grease.AppName = "enumgen"
	grease.AppTitle = "Enumgen"
	grease.AppAbout = "Enumgen is a tool that generates helpful methods for Go enums."
	grease.DefaultFiles = []string{"enumgen.toml"}
	grease.Run(&App{}, &enumgen.Config{})
}
