// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"os"
)

var (
	// AppName is the internal name of the Grease app
	// (typically in kebab-case) (see also [AppTitle])
	AppName = "grease"

	// AppTitle is the user-visible name of the Grease app
	// (typically in Title Case) (see also [AppName])
	AppTitle = "Grease"

	// AppAbout is the description of the Grease app
	AppAbout = "Grease allows you to edit configuration information and run commands through a CLI and a GUI interface."

	// Fatal is whether to, if there is an error in [Run],
	// print it and fatally exit the program with [os.Exit]
	// and an exit code of 1.
	Fatal = true
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
func Run[T any, C CmdOrFunc[T]](cfg T, cmds ...C) error {
	cs, err := CmdsFromCmdOrFuncs[T, C](cmds)
	if err != nil {
		err := fmt.Errorf("error getting commands from given commands: %w", err)
		if Fatal {
			fmt.Println(err)
			os.Exit(1)
		}
		return err
	}
	leftovers, err := Config(cfg, "", cs...)
	if err != nil {
		err := fmt.Errorf("error configuring app: %w", err)
		if Fatal {
			fmt.Println(err)
			os.Exit(1)
		}
		return err
	}
	// root command if no other command is specified
	cmd := ""
	if len(leftovers) > 0 {
		cmd = leftovers[0]
	}
	err = RunCmd(cfg, cmd, cs...)
	if err != nil {
		err := fmt.Errorf("error running command %q: %w", cmd, err)
		if Fatal {
			fmt.Println(err)
			os.Exit(1)
		}
		return err
	}
	return nil
}

// RunCmd runs the command with the given
// name on the given app. It looks for the
// method with the name of the command converted
// to camel case suffixed with "Cmd"; for example,
// for a command named "build", it will look for a
// method named "BuildCmd".
func RunCmd[T any](cfg T, cmd string, cmds ...Cmd[T]) error {
	for _, c := range cmds {
		if c.Name == cmd {
			err := c.Func(cfg)
			if err != nil {
				return err
			}
			return nil
		}
	}
	if cmd == "" || cmd == "help" {
		fmt.Println(Usage(cfg, cmds...))
		return nil
	}
	return fmt.Errorf("command %q not found", cmd)
}
