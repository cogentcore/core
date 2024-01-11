// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/goki/gokitool/cmd"
	"goki.dev/goki/gokitool/config"
	"goki.dev/goki/gokitool/generate"
	"goki.dev/goki/grease"
)

func main() {
	opts := grease.DefaultOptions("goki", "Goki", "Command line and GUI tools for developing apps and libraries using the Goki framework.")
	opts.DefaultFiles = []string{".goki/config.toml"}
	opts.SearchUp = true
	grease.Run(opts, &config.Config{}, cmd.Build, cmd.Install, cmd.Run, generate.Generate, cmd.Init, cmd.Pack, cmd.Setup, cmd.Log, cmd.VersionRelease, cmd.Release, cmd.GetVersion, cmd.SetVersion, cmd.UpdateVersion)
}
