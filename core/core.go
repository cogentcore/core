// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/core/cmd"
	"cogentcore.org/core/core/config"
	"cogentcore.org/core/core/generate"
	"cogentcore.org/core/grease"
)

func main() {
	opts := grease.DefaultOptions("Cogent Core", "Command line tools for developing apps and libraries using the Cogent Core framework.")
	opts.DefaultFiles = []string{"core.toml"}
	opts.SearchUp = true
	grease.Run(opts, &config.Config{}, cmd.Setup, cmd.Build, cmd.Run, cmd.Pack, cmd.Install, generate.Generate, cmd.Changed, cmd.Pull, cmd.Log, cmd.Release, cmd.NextRelease)
}
