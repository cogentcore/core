// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// The color functions that grease uses internally;
// they can be used and modified externally to
// achieve and maintain a good and consistent color scheme.
var (
	// ErrorColor is the color printer function for errors
	ErrorColor = color.New(color.FgYellow).SprintfFunc()
	// SuccessColor is the color printer function for success messages
	SuccessColor = color.New(color.FgGreen, color.Bold).SprintfFunc()
	// CmdColor is the color printer function for commands, arguments, and flags
	CmdColor = color.New(color.FgCyan, color.Bold).SprintfFunc()
)

// Run runs an app with the given options, configuration struct,
// and commands. It does not run the GUI; see [goki.dev/greasi.Run]
// for that. The configuration struct should be passed as a pointer, and
// configuration options should be defined as fields on the configuration
// struct. The commands can be specified as either functions or struct
// objects; the functions are more concise but require using gti
// (see https://goki.dev/gti). In addition to the given commands,
// Run adds a "help" command that prints the result of [Usage], which
// will also be the root command if no other root command is specified.
// Also, it adds the fields in [MetaConfig] as configuration options.
// If [Options.Fatal] is set to true, the error result of Run does
// not need to be handled. Run uses [os.Args] for its arguments.
func Run[T any, C CmdOrFunc[T]](opts *Options, cfg T, cmds ...C) error {
	cs, err := CmdsFromCmdOrFuncs[T, C](cmds)
	if err != nil {
		err := fmt.Errorf("error getting commands from given commands: %w", err)
		if opts.Fatal {
			fmt.Println(ErrorColor("%v", err))
			os.Exit(1)
		}
		return err
	}
	cmd, err := Config(opts, cfg, cs...)
	if err != nil {
		err := fmt.Errorf("error configuring app: %w", err)
		if opts.Fatal {
			fmt.Println(ErrorColor("%v", err))
			os.Exit(1)
		}
		return err
	}
	err = RunCmd(opts, cfg, cmd, cs...)
	if err != nil {
		if opts.Fatal {
			fmt.Println(CmdColor(cmdString(opts, cmd)) + ErrorColor(" failed: %v", err))
			os.Exit(1)
		}
		return fmt.Errorf("%s failed: %w", opts.AppName+" "+cmd, err)
	}
	if opts.PrintSuccess {
		fmt.Println(CmdColor(cmdString(opts, cmd)) + SuccessColor(" succeeded"))
	}
	return nil
}

// RunCmd runs the command with the given name using the given options,
// configuration information, and available commands. If the given
// command name is "", it runs the root command.
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
	if cmd == "" { // if we couldn't find the command and we are looking for the root command, we fall back on help
		fmt.Println(Usage(opts, cfg, cmd, cmds...))
		os.Exit(0)
	}
	return fmt.Errorf("command %q not found", cmd)
}

// cmdString is a simple helper function that returns a string
// with [Options.AppName] and the given command name string combined
// to form a string representing the complete command being run.
func cmdString(opts *Options, cmd string) string {
	if cmd == "" {
		return opts.AppName
	}
	return opts.AppName + " " + cmd
}
