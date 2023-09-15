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
	// NonFlagArgs are the command-line args that remain after all the flags have
	// been processed.  This is set after the call to Config.
	NonFlagArgs = []string{}

	// ConfigFile is the name of the config file actually loaded, specified by the
	// -config or -cfg command-line arg or the default file given in Config
	ConfigFile string

	// Help is variable target for -help or -h args
	Help bool
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
func Config[T any](opts *Options, cfg T, cmd string, cmds ...*Cmd[T]) ([]string, error) {
	var errs []error
	err := SetFromDefaults(cfg)
	if err != nil {
		errs = append(errs, err)
	}

	// first, we do a simplified pass to get the meta
	// command flags (help and config), which we need
	// to know before we can do other configuration.
	allArgs := make(map[string]reflect.Value)
	CommandFlags(allArgs)

	args := os.Args[1:]
	_, flags, err := GetArgs(args)
	if err != nil {
		errs = append(errs, err)
	}
	err = ParseFlags(flags, allArgs, false) // false = ignore non-matches, because we are only handling meta flags, not all of them
	if err != nil {
		errs = append(errs, err)
	}

	if Help {
		fmt.Println(Usage(opts, cfg, cmd, cmds...))
		os.Exit(0)
	}

	var cfgFiles []string
	if ConfigFile != "" {
		_, err := dirs.FindFileOnPaths(opts.IncludePaths, ConfigFile)
		if err == nil {
			cfgFiles = append(cfgFiles, ConfigFile)
		}
	} else {
		if opts.SearchUp {
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
				opts.IncludePaths = append(opts.IncludePaths, wd)
			}
		}
		for _, fn := range opts.DefaultFiles {
			_, err := dirs.FindFileOnPaths(opts.IncludePaths, fn)
			if err == nil {
				cfgFiles = append(cfgFiles, fn)
			}
		}
	}

	if opts.NeedConfigFile && len(cfgFiles) == 0 {
		err = errors.New("grease.Config: no config file or default files specified")
		return nil, err
	}

	for _, fn := range cfgFiles {
		err = OpenWithIncludes(opts, cfg, fn)
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
