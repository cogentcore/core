// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"fmt"
	"os"
	"os/exec"
)

// LookPath searches for an executable named file in the
// directories named by the PATH environment variable.
// If file contains a slash, it is tried directly and the PATH is not consulted.
// Otherwise, on success, the result is an absolute path.
//
// In older versions of Go, LookPath could return a path relative to the current directory.
// As of Go 1.19, LookPath will instead return that path along with an error satisfying
// errors.Is(err, ErrDot). See the package documentation for more details.
func LookPath(file string) (string, error) { return exec.LookPath(file) }

// RemoveAll is a helper function that calls [os.RemoveAll] and [Config.PrintCmd].
func (c *Config) RemoveAll(path string) error {
	var err error
	if !c.PrintOnly {
		err = os.RemoveAll(path)
	}
	c.PrintCmd(fmt.Sprintf("rm -rf %q", path), err)
	return err
}

// RemoveAll calls [Config.RemoveAll] on [Major]
func RemoveAll(path string) error {
	return Major().RemoveAll(path)
}

// MkdirAll is a helper function that calls [os.MkdirAll] and [Config.PrintCmd].
func (c *Config) MkdirAll(path string, perm os.FileMode) error {
	var err error
	if !c.PrintOnly {
		err = os.MkdirAll(path, perm)
	}
	c.PrintCmd(fmt.Sprintf("mkdir -p %q", path), err)
	return err
}

// MkdirAll calls [Config.MkdirAll] on [Major]
func MkdirAll(path string, perm os.FileMode) error {
	return Major().MkdirAll(path, perm)
}
