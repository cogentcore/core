// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"goki.dev/gear"
	"goki.dev/goki/cmd"
)

func main() {
	gear.AppName = "goki"
	gear.AppAbout = "Command line and GUI tools for developing apps and libraries using the GoKi framework."
	err := gear.Run(cmd.TheApp, ".goki/config.toml")
	if err != nil {
		fmt.Println(err)
	}
}
