// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"fmt"
	"io/fs"

	"github.com/goki/ki/toml"
)

// OpenWithIncludes reads config from given config file,
// looking on IncludePaths for the file,
// and opens any Includes specified in the given config file
// in the natural include order so includee overwrites included settings.
// Is equivalent to Open if there are no Includes.
// Returns an error if any of the include files cannot be found on IncludePath.
func OpenWithIncludes(cfg any, file string) error {
	err := toml.OpenFromPaths(cfg, file, IncludePaths)
	if err != nil {
		return err
	}
	incfg, ok := cfg.(Includer)
	if !ok {
		return err
	}
	incs, err := IncludeStack(incfg)
	ni := len(incs)
	if ni == 0 {
		return err
	}
	for i := ni - 1; i >= 0; i-- {
		inc := incs[i]
		err = toml.OpenFromPaths(cfg, inc, IncludePaths)
		if err != nil {
			fmt.Println(err)
		}
	}
	// reopen original
	toml.OpenFromPaths(cfg, file, IncludePaths)
	*incfg.IncludesPtr() = incs
	return err
}

// OpenFS reads config from given TOML file,
// using the fs.FS filesystem -- e.g., for embed files.
func OpenFS(cfg any, fsys fs.FS, file string) error {
	return toml.OpenFS(cfg, fsys, file)
}

// Save writes TOML to given file.
func Save(cfg any, file string) error {
	return toml.Save(cfg, file)
}
