// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package goki provides general functions
// for developing apps and libraries using
// the GoKi framework.
package tools

import (
	"fmt"
	"os"
)

// Init initializes the ".goki" directory
// and a "config.toml" file inside it.
func Init() error {
	err := os.Mkdir(".goki", 0750)
	if err != nil {
		return fmt.Errorf("error creating %q directory: %w", ".goki", err)
	}
	err = os.WriteFile(".goki/config.toml", []byte(`version = "v0.0.0"`), 0666)
	if err != nil {
		return fmt.Errorf("error writing to configuration file: %w", err)
	}
	return nil
}
