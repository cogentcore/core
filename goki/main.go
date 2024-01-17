// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/goki/cmd"
	"cogentcore.org/core/goki/config"
	"cogentcore.org/core/goki/generate"
	"cogentcore.org/core/grease"
)

func main() {
	opts := grease.DefaultOptions("goki", "CogentCore", "Command line and GUI tools for developing apps and libraries using the Cogent Core framework.")
	opts.DefaultFiles = []string{".goki/config.toml"}
	opts.SearchUp = true
	grease.Run(opts, &config.Config{}, cmd.Build, cmd.Install, cmd.Run, generate.Generate, cmd.Init, cmd.Pack, cmd.Setup, cmd.Log, cmd.VersionRelease, cmd.Release, cmd.GetVersion, cmd.SetVersion, cmd.UpdateVersion)
}
