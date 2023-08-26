// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"encoding/json"
	"fmt"
	"os"
)

// Package contains all of the information about a package.
type Package struct {
	ID              string   // the unique name of the package (ex: dev)
	Name            string   // the user friendly name of the package (ex: The GoKi Developer Tools)
	InstallCommands Commands // the set of commands to run for each operating system to install the package
}

// LoadPackages loads all of the packages from the packages.json file
func LoadPackages() ([]Package, error) {
	// exe, err := os.Executable()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to locate executable directory: %w", err)
	// }
	// exePath := filepath.Dir(exe)
	b, err := os.ReadFile("packages.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load packages file: %w", err)
	}
	var res []Package
	err = json.Unmarshal(b, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal packages: %w", err)
	}
	return res, nil
}
