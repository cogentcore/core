// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"strings"

	"cogentcore.org/core/core/config"
	"cogentcore.org/core/grease"
	"cogentcore.org/core/xe"
	"github.com/Masterminds/semver/v3"
)

// GetVersion prints the version of the project.
func GetVersion(c *config.Config) error { //gti:add
	fmt.Println(c.Version)
	return nil
}

// SetVersion updates the config file based on the version and commits and pushes
// the changes. After it, release or similar should be called to push the git tags.
func SetVersion(c *config.Config) error { //gti:add
	// we need to update the config file with the new version
	// TODO: determine correct config file instead of just first one
	err := grease.Save(c, grease.ConfigFiles[0])
	if err != nil {
		return fmt.Errorf("error saving new version to config file: %w", err)
	}
	err = PushVersionGit(c)
	if err != nil {
		return fmt.Errorf("error pushing version file to Git: %w", err)
	}
	return nil
}

// UpdateVersion updates the version of the project by one patch version.
// After it, release or similar should be called to push the git tags.
func UpdateVersion(c *config.Config) error { //gti:add
	ver, err := semver.NewVersion(c.Version)
	if err != nil {
		return fmt.Errorf("error getting semver version from version %q: %w", c.Version, err)
	}

	if !strings.HasPrefix(ver.Prerelease(), "dev") { // if no dev pre-release, we can just do standard increment
		*ver = ver.IncPatch()
	} else { // otherwise, we have to increment pre-release version instead
		pvn := strings.TrimPrefix(ver.Prerelease(), "dev")
		pver, err := semver.NewVersion(pvn)
		if err != nil {
			return fmt.Errorf("error parsing dev version %q from version %q: %w", pvn, c.Version, err)
		}
		*pver = pver.IncPatch()
		// apply incremented pre-release version to main version
		nv, err := ver.SetPrerelease("dev" + pver.String())
		if err != nil {
			return fmt.Errorf("error setting pre-release of new version to %q from repository version %q: %w", "dev"+pver.String(), c.Version, err)
		}
		*ver = nv
	}

	c.Version = "v" + ver.String()
	return SetVersion(c) // now we can set to newly calculated version
}

// PushVersionGit makes and pushes a Git commit updating the version based on the given
// config info. It does not actually update the version; it only commits and pushes the
// changes that should have already been made by [UpdateVersion].
func PushVersionGit(c *config.Config) error {
	err := xe.Run("git", "commit", "-am", "updated version to "+c.Version)
	if err != nil {
		return fmt.Errorf("error commiting release commit: %w", err)
	}
	err = xe.Run("git", "push")
	if err != nil {
		return fmt.Errorf("error pushing commit: %w", err)
	}
	return nil
}
