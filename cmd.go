// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command core provides command line tools for developing apps
// and libraries using the Cogent Core framework.
package main

import (
	"cogentcore.org/core/cli"
	"cogentcore.org/core/cmd/core/cmd"
	"cogentcore.org/core/cmd/core/config"
	"cogentcore.org/core/cmd/core/generate"
)

func main() {
	opts := cli.DefaultOptions("Cogent Core", "Command line tools for developing apps and libraries using the Cogent Core framework.")
	opts.DefaultFiles = []string{"core.toml"}
	opts.SearchUp = true
	cli.Run(opts, &config.Config{}, cmd.Setup, cmd.Build, cmd.Run, cmd.Pack, cmd.Install, generate.Generate, cmd.Changed, cmd.Pull, cmd.Log, cmd.Release, cmd.NextRelease)
}
