// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"goki.dev/gear"
)

type App struct {

	// the name of the app
	Name string `desc:"the name of the app"`

	// the version of the app
	Version string `desc:"the version of the app"`
}

var TheApp = &App{}

func main() {
	gear.AppName = "goki"
	gear.AppAbout = "Command line and GUI tools for developing apps and libraries using the GoKi framework."
	err := gear.Run(TheApp, ".goki/config.toml")
	if err != nil {
		fmt.Println(err)
	}
}
