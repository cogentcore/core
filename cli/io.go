// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"io/fs"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/reflectx"
)

// OpenWithIncludes reads the config struct from the given config file
// using the given options, looking on [Options.IncludePaths] for the file.
// It opens any Includes specified in the given config file in the natural
// include order so that includers overwrite included settings.
// Is equivalent to Open if there are no Includes. It returns an error if
// any of the include files cannot be found on [Options.IncludePaths].
func OpenWithIncludes(opts *Options, cfg any, file string) error {
	files := dirs.FindFilesOnPaths(opts.IncludePaths, file)
	if len(files) == 0 {
		return fmt.Errorf("OpenWithIncludes: no files found for %q", file)
	}
	err := tomlx.OpenFiles(cfg, files)
	if err != nil {
		return err
	}
	incfg, ok := cfg.(Includer)
	if !ok {
		return err
	}
	incs, err := IncludeStack(opts, incfg)
	ni := len(incs)
	if ni == 0 {
		return err
	}
	for i := ni - 1; i >= 0; i-- {
		inc := incs[i]
		err = tomlx.OpenFiles(cfg, dirs.FindFilesOnPaths(opts.IncludePaths, inc))
		if err != nil {
			fmt.Println(err)
		}
	}
	// reopen original
	err = tomlx.OpenFiles(cfg, dirs.FindFilesOnPaths(opts.IncludePaths, file))
	if err != nil {
		return err
	}
	*incfg.IncludesPtr() = incs
	return err
}

// OpenFS reads the given config object from the given file.
func Open(cfg any, file string) error {
	return tomlx.Open(cfg, file)
}

// OpenFS reads the given config object from given file, using
// the given [fs.FS] filesystem (e.g., for embed files).
func OpenFS(cfg any, fsys fs.FS, file string) error {
	return tomlx.OpenFS(cfg, fsys, file)
}

// Save writes the given config object to the given file.
// It only saves the non-default fields of the given object,
// as specified by [reflectx.NonDefaultFields].
func Save(cfg any, file string) error {
	return tomlx.Save(reflectx.NonDefaultFields(cfg), file)
}
