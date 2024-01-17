// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate

import (
	"cogentcore.org/core/gear"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/grease"
)

type config struct { //gti:add
	// Command is the command to run gear on
	Command string `posarg:"0" required:"-" def:"ls"`
}

func main() {
	opts := grease.DefaultOptions("gear", "Gear",
		"Gear provides the generation of GUIs and interactive CLIs for any existing command line tools.")
	grease.Run(opts, &config{}, &grease.Cmd[*config]{
		Func: run,
		Root: true,
	})
}

func run(c *config) error {
	b := gi.NewAppBody("Gear")
	cmd := gear.NewCmd(c.Command)
	err := cmd.Parse()
	if err != nil {
		return err
	}
	app := gear.NewApp(b).SetCmd(cmd)
	b.AddAppBar(app.AppBar)
	b.NewWindow().Run().Wait()
	return nil
}
