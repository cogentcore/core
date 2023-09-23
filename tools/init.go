// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"goki.dev/goki/config"
	"goki.dev/grows/tomls"
)

// Init initializes the ".goki" directory
// and a "config.toml" file inside it.
// The "config.toml" file has the given
// config info. Init also sets the config name
// to the current directory if it is unset.
//
//gti:add
func Init(c *config.Config) error {
	err := os.Mkdir(".goki", 0750)
	if err != nil {
		return fmt.Errorf("error creating %q directory: %w", ".goki", err)
	}
	if c.Name == "" {
		cdir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error finding current directory: %w", err)
		}
		base := filepath.Base(cdir)
		c.Name = base
	}
	err = tomls.Save(c, ".goki/config.toml")
	if err != nil {
		return fmt.Errorf("error writing to configuration file: %w", err)
	}
	return nil
}
