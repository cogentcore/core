// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"fmt"

	"goki.dev/goki/config"
)

// Run builds and runs the config package. It uses the same
// config info as build.
func Run(c *config.Config) error {
	err := Build(c)
	if err != nil {
		return fmt.Errorf("error building package: %w", err)
	}
	return nil
}
