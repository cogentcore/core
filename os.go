// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xe

import (
	"fmt"
	"os"
)

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
