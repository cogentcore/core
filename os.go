// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xe

import (
	"fmt"
	"os"
)

// RemoveAll is a simple helper function that calls [os.RemoveAll] and [Config.PrintCmd].
func (c *Config) RemoveAll(path string) error {
	err := os.RemoveAll(path)
	c.PrintCmd(fmt.Sprintf("rm -rf %q", path), err)
	return err
}
