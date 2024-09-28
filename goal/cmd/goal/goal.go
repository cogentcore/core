// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command goal is an interactive cli for running and compiling Goal code.
package main

import (
	"cogentcore.org/core/cli"
	"cogentcore.org/core/goal/interpreter"
)

func main() { //types:skip
	opts := cli.DefaultOptions("goal", "An interactive tool for running and compiling Goal (Go augmented language).")
	cfg := &interpreter.Config{}
	cfg.InteractiveFunc = interpreter.Interactive
	cli.Run(opts, cfg, interpreter.Run, interpreter.Build)
}
