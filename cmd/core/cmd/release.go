// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"strings"

	"cogentcore.org/core/cmd/core/config"
	"cogentcore.org/core/xe"
	"github.com/Masterminds/semver/v3"
)

// Release releases the project with the specified git version tag.
func Release(c *config.Config) error { //gti:add
	err := xe.Run("git", "tag", "-a", c.Version, "-m", c.Version+" release")
	if err != nil {
		return fmt.Errorf("error tagging release: %w", err)
	}
	err = xe.Run("git", "push", "origin", "--tags")
	if err != nil {
		return fmt.Errorf("error pushing tags: %w", err)
	}
	return nil
}

// NextRelease releases the project with the current git version
// tag incremented by one patch version.
func NextRelease(c *config.Config) error { //gti:add
	ver, err := NextVersion(c)
	if err != nil {
		return err
	}
	c.Version = ver
	return Release(c)
}

// NextVersion returns the version of the project
// incremented by one patch version.
func NextVersion(c *config.Config) (string, error) {
	cur, err := xe.Output("git", "describe", "--tags", "--abbrev=0")
	if err != nil {
		return "", err
	}
	ver, err := semver.NewVersion(cur)
	if err != nil {
		return "", fmt.Errorf("error getting semver version from version %q: %w", c.Version, err)
	}

	if !strings.HasPrefix(ver.Prerelease(), "dev") { // if no dev pre-release, we can just do standard increment
		*ver = ver.IncPatch()
	} else { // otherwise, we have to increment pre-release version instead
		pvn := strings.TrimPrefix(ver.Prerelease(), "dev")
		pver, err := semver.NewVersion(pvn)
		if err != nil {
			return "", fmt.Errorf("error parsing dev version %q from version %q: %w", pvn, c.Version, err)
		}
		*pver = pver.IncPatch()
		// apply incremented pre-release version to main version
		nv, err := ver.SetPrerelease("dev" + pver.String())
		if err != nil {
			return "", fmt.Errorf("error setting pre-release of new version to %q from repository version %q: %w", "dev"+pver.String(), c.Version, err)
		}
		*ver = nv
	}
	return "v" + ver.String(), nil
}
