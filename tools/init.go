// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tools

import (
	"fmt"
	"os"

	"goki.dev/goki/config"
	"goki.dev/grows/tomls"
)

// Init initializes the ".goki" directory
// and a "config.toml" file inside it.
// The "config.toml" file has the given
// config info. Init also sets the config name
// to the current directory if it is unset.
func Init(c *config.Config) error { //gti:add
	err := os.MkdirAll(".goki", 0750)
	if err != nil {
		return fmt.Errorf("error creating %q directory: %w", ".goki", err)
	}
	err = tomls.Save(c, ".goki/config.toml")
	if err != nil {
		return fmt.Errorf("error writing to configuration file: %w", err)
	}
	return nil
}
