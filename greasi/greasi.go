// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package greasi

import (
	"fmt"
	"os"

	"cogentcore.org/core/cli"
	"cogentcore.org/core/grog"
)

// Run runs the given app with the given default
// configuration file paths. It is similar to
// [cli.Run], but it also runs the GUI if no
// arguments were provided. The app should be
// a pointer, and configuration options should
// be defined as fields on the app type. Also,
// commands should be defined as methods on the
// app type with the suffix "Cmd"; for example,
// for a command named "build", there should be
// the method:
//
//	func (a *App) BuildCmd() error
//
// Run uses [os.Args] for its arguments.
func Run[T any, C cli.CmdOrFunc[T]](opts *cli.Options, cfg T, cmds ...C) error {
	cs, err := cli.CmdsFromCmdOrFuncs[T, C](cmds)
	if err != nil {
		err := fmt.Errorf("error getting commands from given commands: %w", err)
		if opts.Fatal {
			grog.PrintlnError(err)
			os.Exit(1)
		}
		return err
	}
	cs = cli.AddCmd(cs, &cli.Cmd[T]{
		Func: func(t T) error {
			GUI(opts, t, cs...)
			return nil
		},
		Name: "gui",
		Doc:  "gui runs the GUI version of the " + opts.AppName + " tool",
		Root: true, // if root isn't already taken, we take it
	})
	return cli.Run(opts, cfg, cs...)
}
