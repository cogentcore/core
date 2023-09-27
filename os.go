// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xe

import (
	"fmt"
	"os"
)

// RemoveAll is a simple helper function that calls [os.RemoveAll]
// after printing an equivalent command to [Config.Commands].
func (c *Config) RemoveAll(path string) error {
	err := os.RemoveAll(path)
	cmds := c.GetWriter(c.Commands, err)
	if cmds != nil {
		fmt.Fprintf(cmds, "rm -rf %q\n", path)
	}
	return err
}
