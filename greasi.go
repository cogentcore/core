// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package greasi

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"goki.dev/grease"
)

// Run runs the given app with the given default
// configuration file paths. It is similar to
// [grease.Run], but it also runs the GUI if no
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
func Run[T any, C grease.CmdOrFunc[T]](opts *grease.Options, cfg T, cmds ...C) error {
	cs, err := grease.CmdsFromCmdOrFuncs[T, C](cmds)
	if err != nil {
		err := fmt.Errorf("error getting commands from given commands: %w", err)
		if opts.Fatal {
			color.Red("%v", err)
			os.Exit(1)
		}
		return err
	}
	cs = grease.AddCmd(cs, &grease.Cmd[T]{
		Func: func(t T) error {
			GUI(opts, t, cs...)
			return nil
		},
		Name: "gui",
		Doc:  "runs the GUI version of the " + opts.AppTitle + " tool",
		Root: true, // if root isn't already taken, we take it
	})
	return grease.Run(opts, cfg, cs...)
}
