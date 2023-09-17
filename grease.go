// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// color functions for internal use
var (
	errorColor   = color.New(color.FgRed).SprintfFunc()
	successColor = color.New(color.FgGreen).SprintfFunc()
	cmdColor     = color.New(color.FgCyan).SprintfFunc()
)

// Run runs the given app with the given default
// configuration file paths. It does not run the
// GUI; see [goki.dev/greasi.Run] for that. The app should be
// a pointer, and configuration options should
// be defined as fields on the app type. Also,
// commands should be defined as methods on the
// app type with the suffix "Cmd"; for example,
// for a command named "build", there should be
// the method:
//
//	func (a *App) BuildCmd() error
//
// If no command is provided, Run calls the
// method "RootCmd" if it exists. If it does
// not exist, or the command provided is "help"
// and "HelpCmd" does not exist, Run prints
// the result of [Usage].
// Run uses [os.Args] for its arguments.
func Run[T any, C CmdOrFunc[T]](opts *Options, cfg T, cmds ...C) error {
	cs, err := CmdsFromCmdOrFuncs[T, C](cmds)
	if err != nil {
		err := fmt.Errorf("error getting commands from given commands: %w", err)
		if opts.Fatal {
			fmt.Println(errorColor("%v", err))
			os.Exit(1)
		}
		return err
	}
	cmd, err := Config(opts, cfg, cs...)
	if err != nil {
		err := fmt.Errorf("error configuring app: %w", err)
		if opts.Fatal {
			fmt.Println(errorColor("%v", err))
			os.Exit(1)
		}
		return err
	}
	err = RunCmd(opts, cfg, cmd, cs...)
	if err != nil {
		if opts.Fatal {
			fmt.Println(cmdColor(cmdString(opts, cmd)) + errorColor(" failed: %v", err))
			os.Exit(1)
		}
		return fmt.Errorf("%s failed: %w", opts.AppName+" "+cmd, err)
	}
	if opts.PrintSuccess {
		fmt.Println(cmdColor(cmdString(opts, cmd)) + successColor(" succeeded"))
	}
	return nil
}

// cmdString is a simple helper function that
// returns a string with [Options.AppName]
// and the given command name string.
func cmdString(opts *Options, cmd string) string {
	if cmd == "" {
		return opts.AppName
	}
	return opts.AppName + " " + cmd
}

// RunCmd runs the command with the given
// name on the given app. It looks for the
// method with the name of the command converted
// to camel case suffixed with "Cmd"; for example,
// for a command named "build", it will look for a
// method named "BuildCmd".
func RunCmd[T any](opts *Options, cfg T, cmd string, cmds ...*Cmd[T]) error {
	for _, c := range cmds {
		if c.Name == cmd || c.Root && cmd == "" {
			err := c.Func(cfg)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("command %q not found", cmd)
}
