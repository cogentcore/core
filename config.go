// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"goki.dev/ki/v2/dirs"
	"goki.dev/ki/v2/kit"
)

var (
	// DefaultEncoding is the default encoding format for config files.
	// currently toml is the only supported format, but others could be added
	// if needed.
	DefaultEncoding = "toml"

	// DefaultFiles are the default configuration file paths
	DefaultFiles []string = []string{"config.toml"}

	// IncludePaths is a list of file paths to try for finding config files
	// specified in Include field or via the command line --config --cfg or -c args.
	// Set this prior to calling Config -- default is current directory '.' and 'configs'
	IncludePaths = []string{".", "configs"}

	// NonFlagArgs are the command-line args that remain after all the flags have
	// been processed.  This is set after the call to Config.
	NonFlagArgs = []string{}

	// ConfigFile is the name of the config file actually loaded, specified by the
	// -config or -cfg command-line arg or the default file given in Config
	ConfigFile string

	// Help is variable target for -help or -h args
	Help bool

	// SearchUp indicates whether to search up the filesystem
	// for the default config file by checking the provided default
	// config file location relative to each directory up the tree
	SearchUp bool

	// NeedConfigFile indicates whether a configuration file
	// must be provided for the command to run
	NeedConfigFile bool
)

// Config is the overall config setting function, processing config files
// and command-line arguments, in the following order:
//   - Apply any `def:` field tag default values.
//   - Look for `--config`, `--cfg`, or `-c` arg, specifying a config file on the command line.
//   - Fall back on default config file name passed to `Config` function, if arg not found.
//   - Read any `Include[s]` files in config file in deepest-first (natural) order,
//     then the specified config file last.
//   - if multiple config files are listed, then the first one that exists is used
//   - Process command-line args based on Config field names, with `.` separator
//     for sub-fields.
//   - Boolean flags are set on with plain -flag; use No prefix to turn off
//     (or explicitly set values to true or false).
//
// Also processes -help or -h and prints usage and quits immediately.
// Config uses [os.Args] for its arguments.
func Config[T any](cfg T, cmd string, cmds ...Cmd[T]) ([]string, error) {
	var errs []error
	err := SetFromDefaults(cfg)
	if err != nil {
		errs = append(errs, err)
	}

	allArgs := make(map[string]reflect.Value)
	CommandArgs(allArgs)

	args := os.Args[1:]
	_, err = ParseArgs(cfg, args, allArgs, false) // false = ignore non-matches
	if err != nil {
		errs = append(errs, err)
	}

	if Help {
		fmt.Println(Usage(cfg, cmds...))
		os.Exit(0)
	}

	var cfgFiles []string
	if ConfigFile != "" {
		_, err := dirs.FindFileOnPaths(IncludePaths, ConfigFile)
		if err == nil {
			cfgFiles = append(cfgFiles, ConfigFile)
		}
	} else {
		if SearchUp {
			wd, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("error getting current directory: %w", err)
			}
			pwd := wd
			for {
				pwd = wd
				wd = filepath.Dir(pwd)
				if wd == pwd { // if there is no change, we have reached the root of the filesystem
					break
				}
				IncludePaths = append(IncludePaths, wd)
			}
		}
		for _, fn := range DefaultFiles {
			_, err := dirs.FindFileOnPaths(IncludePaths, fn)
			if err == nil {
				cfgFiles = append(cfgFiles, fn)
			}
		}
	}

	if NeedConfigFile && len(cfgFiles) == 0 {
		err = errors.New("grease.Config: no config file or default files specified")
		return nil, err
	}

	for _, fn := range cfgFiles {
		err = OpenWithIncludes(cfg, fn)
		if err != nil {
			errs = append(errs, err)
		}
	}

	NonFlagArgs, err = SetFromArgs(cfg, args)
	if err != nil {
		errs = append(errs, err)
	}
	return NonFlagArgs, kit.AllErrors(errs, 10)
}
