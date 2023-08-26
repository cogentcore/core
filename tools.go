// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

// Init initializes the ".goki" directory
// and a "config.toml" file inside it.
func Init() error {
	err := os.Mkdir(".goki", 0750)
	if err != nil {
		return fmt.Errorf("error creating %q directory: %w", ".goki", err)
	}
	cdir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error finding current directory: %w", err)
	}
	base := filepath.Base(cdir)
	err = os.WriteFile(".goki/config.toml", []byte(`name = "`+base+`"
version = "v0.0.0"
`), 0666)
	if err != nil {
		return fmt.Errorf("error writing to configuration file: %w", err)
	}
	return nil
}
