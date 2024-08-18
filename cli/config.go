// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/logx"
)

// TheMetaConfig holds the current [MetaConfig] data for access
// of verbose and quiet state from client apps.
var TheMetaConfig *MetaConfig

// IMPORTANT: all changes to [metaConfig] must be updated in [metaConfigFields]

// MetaConfig contains meta configuration information specified
// via command line arguments that controls the initial behavior
// of cli for all apps before anything else is loaded. Its
// main purpose is to support the help command and flag and
// the specification of custom config files on the command line.
type MetaConfig struct {
	// the file name of the config file to load
	Config string `flag:"cfg,config"`

	// whether to display a help message
	Help bool `flag:"h,help"`

	// the name of the command to display
	// help information for. It is only applicable to the
	// help command, but it is enabled for all commands so
	// that it can consume all positional arguments to prevent
	// errors about unused arguments.
	HelpCmd string `posarg:"all"`

	// whether to run the command in verbose mode
	// and print more information
	Verbose bool `flag:"v,verbose"`

	// whether to run the command in very verbose mode
	// and print as much information as possible
	VeryVerbose bool `flag:"vv,very-verbose"`

	// whether to run the command in quiet mode
	// and print less information
	Quiet bool `flag:"q,quiet"`
}

// metaConfigFields is the struct used for the implementation
// of [addMetaConfigFields], and for the usage information for
// meta configuration options in [usage].
// NOTE: we could do this through [MetaConfig], but that
// causes problems with the HelpCmd field capturing
// everything, so it easier to just add through a separate struct.
// TODO: maybe improve the structure of this.
// TODO: can we get HelpCmd to display correctly in usage?
type metaConfigFields struct { //types:add
	// the file name of the config file to load
	Config string `flag:"cfg,config"`

	// whether to display a help message
	Help bool `flag:"h,help"`

	// the name of the command to display
	// help information for.
	HelpCmd string `cmd:"help" posarg:"all"`

	// whether to run the command in verbose mode
	// and print more information
	Verbose bool `flag:"v,verbose"`

	// whether to run the command in very verbose mode
	// and print as much information as possible
	VeryVerbose bool `flag:"vv,very-verbose"`

	// whether to run the command in quiet mode
	// and print less information
	Quiet bool `flag:"q,quiet"`
}

// addMetaConfigFields adds meta fields that control the config process
// to the given map of fields. These fields have no actual effect and
// map to a placeholder value because they are handled elsewhere, but
// they must be set to prevent errors about missing flags. The flags
// that it adds are those in [metaConfig].
func addMetaConfigFields(allFields *fields) {
	addFields(&metaConfigFields{}, allFields, "")
}

// metaCmds is a set of commands based on [MetaConfig] that
// contains a shell implementation of the help command.
var metaCmds = []*Cmd[*MetaConfig]{
	{
		Func: func(mc *MetaConfig) error { return nil }, // this gets handled seperately in [Config], so we don't actually need to do anything here
		Name: "help",
		Doc:  "show usage information for a command",
		Root: true,
	},
}

// OnConfigurer represents a configuration object that specifies a method to
// be called at the end of the [config] function, with the command that has
// been parsed as an argument.
type OnConfigurer interface {
	OnConfig(cmd string) error
}

// config is the main, high-level configuration setting function,
// processing config files and command-line arguments in the following order:
//   - Apply any `default:` field tag default values.
//   - Look for `--config`, `--cfg`, or `-c` arg, specifying a config file on the command line.
//   - Fall back on default config file name passed to `config` function, if arg not found.
//   - Read any `Include[s]` files in config file in deepest-first (natural) order,
//     then the specified config file last.
//   - If multiple config files are found, then they are applied in reverse order, meaning
//     that the first specified file takes the highest precedence.
//   - Process command-line args based on config field names.
//   - Boolean flags are set on with plain -flag; use No prefix to turn off
//     (or explicitly set values to true or false).
//
// config also processes -help and -h by printing the [usage] and quitting immediately.
// It takes [Options] that control its behavior, the configuration struct, which is
// what it sets, and the commands, which it uses for context. Also, it uses [os.Args]
// for its command-line arguments. It returns the command, if any, that was passed in
// [os.Args], and any error that ocurred during the configuration process.
func config[T any](opts *Options, cfg T, cmds ...*Cmd[T]) (string, error) {
	var errs []error
	err := SetFromDefaults(cfg)
	if err != nil {
		errs = append(errs, err)
	}

	args := os.Args[1:]

	// first, we do a pass to get the meta command flags
	// (help and config), which we need to know before
	// we can do other configuration.
	mc := &MetaConfig{}
	// we ignore not found flags in meta config, because we only care about meta config and not anything else being passed to the command
	cmd, err := SetFromArgs(mc, args, NoErrNotFound, metaCmds...)
	if err != nil {
		// if we can't do first set for meta flags, we return immediately (we only do AllErrors for more specific errors)
		return cmd, fmt.Errorf("error doing meta configuration: %w", err)
	}
	TheMetaConfig = mc
	logx.UserLevel = logx.LevelFromFlags(mc.VeryVerbose, mc.Verbose, mc.Quiet)

	// both flag and command trigger help
	if mc.Help || cmd == "help" {
		// string version of args slice has [] on the side, so need to get rid of them
		mc.HelpCmd = strings.TrimPrefix(strings.TrimSuffix(mc.HelpCmd, "]"), "[")
		// if flag and no posargs, will be nil
		if mc.HelpCmd == "nil" {
			mc.HelpCmd = ""
		}
		fmt.Println(usage(opts, cfg, mc.HelpCmd, cmds...))
		os.Exit(0)
	}

	var cfgFiles []string
	if mc.Config != "" {
		cfgFiles = append(cfgFiles, mc.Config)
	}
	cfgFiles = append(cfgFiles, opts.DefaultFiles...)

	if opts.SearchUp {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("error getting current directory: %w", err)
		}
		pwd := wd
		for {
			pwd = wd
			wd = filepath.Dir(pwd)
			if wd == pwd { // if there is no change, we have reached the root of the filesystem
				break
			}
			opts.IncludePaths = append(opts.IncludePaths, wd)
		}
	}

	if opts.NeedConfigFile && len(cfgFiles) == 0 {
		return "", errors.New("cli.Config: no config file or default files specified")
	}

	slices.Reverse(opts.IncludePaths)

	gotAny := false
	for _, fn := range cfgFiles {
		err = openWithIncludes(opts, cfg, fn)
		if err == nil {
			gotAny = true
		}
	}
	if !gotAny && opts.NeedConfigFile {
		return "", errors.New("cli.Config: no config files found")
	}

	cmd, err = SetFromArgs(cfg, args, ErrNotFound, cmds...)
	if err != nil {
		errs = append(errs, err)
	}
	if cfer, ok := any(cfg).(OnConfigurer); ok {
		err := cfer.OnConfig(cmd)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return cmd, errors.Join(errs...)
}
