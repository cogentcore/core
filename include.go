// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// note: FindFileOnPaths adapted from viper package https://github.com/spf13/viper
// Copyright (c) 2014 Steve Francia

package grease

import (
	"reflect"

	"goki.dev/ki/v2/kit"
	"goki.dev/ki/v2/toml"
)

// Includer facilitates processing include files in Config objects.
type Includer interface {
	// IncludesPtr returns a pointer to the Includes []string field containing file(s) to include
	// before processing the current config file.
	IncludesPtr() *[]string
}

// IncludeStack returns the stack of include files in the natural
// order in which they are encountered (nil if none).
// Files should then be read in reverse order of the slice.
// Returns an error if any of the include files cannot be found on IncludePath.
// Does not alter cfg.
func IncludeStack(opts *Options, cfg Includer) ([]string, error) {
	clone := reflect.New(kit.NonPtrType(reflect.TypeOf(cfg))).Interface().(Includer)
	*clone.IncludesPtr() = *cfg.IncludesPtr()
	return includeStackImpl(opts, clone, nil)
}

// includeStackImpl implements IncludeStack, operating on cloned cfg
// todo: could use a more efficient method to just extract the include field..
func includeStackImpl(opts *Options, clone Includer, includes []string) ([]string, error) {
	incs := *clone.IncludesPtr()
	ni := len(incs)
	if ni == 0 {
		return includes, nil
	}
	for i := ni - 1; i >= 0; i-- {
		includes = append(includes, incs[i]) // reverse order so later overwrite earlier
	}
	var errs []error
	for _, inc := range incs {
		*clone.IncludesPtr() = nil
		err := toml.OpenFromPaths(clone, inc, opts.IncludePaths)
		if err == nil {
			includes, err = includeStackImpl(opts, clone, includes)
			if err != nil {
				errs = append(errs, err)
			}
		} else {
			errs = append(errs, err)
		}
	}
	return includes, kit.AllErrors(errs, 10)
}
