// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
)

var (
	// AppName is the internal name of the Grease app
	// (typically in kebab-case) (see also [AppTitle])
	AppName string = "grease"

	// AppTitle is the user-visible name of the Grease app
	// (typically in Title Case) (see also [AppName])
	AppTitle string = "Grease"

	// AppAbout is the description of the Grease app
	AppAbout string = "Grease allows you to edit configuration information and run commands through a CLI and a GUI interface."
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
	leftovers, err := Config(cfg)
	if err != nil {
		return fmt.Errorf("error configuring app: %w", err)
	}
	// root command if no other command is specified
	cmd := ""
	if len(leftovers) > 0 {
		cmd = leftovers[0]
	}
	err = RunCmd(cfg, cmd, cmds...)
	if err != nil {
		return fmt.Errorf("error running command %q: %w", cmd, err)
	}
	return nil
}

// RunCmd runs the command with the given
// name on the given app. It looks for the
// method with the name of the command converted
// to camel case suffixed with "Cmd"; for example,
// for a command named "build", it will look for a
// method named "BuildCmd".
func RunCmd[T any, C CmdOrFunc[T]](cfg T, cmd string, cmds ...C) error {
	cs, err := CmdsFromCmdOrFuncs[T, C](cmds)
	if err != nil {
		return fmt.Errorf("error getting commands from given commands: %w", err)
	}
	for _, c := range cs {
		if c.Name == cmd {
			err := c.Func(cfg)
			if err != nil {
				return fmt.Errorf("error running command %q: %w", c.Name, err)
			}
			return nil
		}
	}
	if cmd == "" || cmd == "help" {
		fmt.Println(Usage(cfg))
		return nil
	}
	return nil
}
