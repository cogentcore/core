// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/iox/tomlx"
)

// openWithIncludes reads the config struct from the given config file
// using the given options, looking on [Options.IncludePaths] for the file.
// It opens any Includes specified in the given config file in the natural
// include order so that includers overwrite included settings.
// Is equivalent to Open if there are no Includes. It returns an error if
// any of the include files cannot be found on [Options.IncludePaths].
func openWithIncludes(opts *Options, cfg any, file string) error {
	files := fsx.FindFilesOnPaths(opts.IncludePaths, file)
	if len(files) == 0 {
		return fmt.Errorf("OpenWithIncludes: no files found for %q", file)
	}
	err := tomlx.OpenFiles(cfg, files...)
	if err != nil {
		return err
	}
	incfg, ok := cfg.(includer)
	if !ok {
		return err
	}
	incs, err := includeStack(opts, incfg)
	ni := len(incs)
	if ni == 0 {
		return err
	}
	for i := ni - 1; i >= 0; i-- {
		inc := incs[i]
		err = tomlx.OpenFiles(cfg, fsx.FindFilesOnPaths(opts.IncludePaths, inc)...)
		if err != nil {
			fmt.Println(err)
		}
	}
	// reopen original
	err = tomlx.OpenFiles(cfg, fsx.FindFilesOnPaths(opts.IncludePaths, file)...)
	if err != nil {
		return err
	}
	*incfg.IncludesPtr() = incs
	return err
}
