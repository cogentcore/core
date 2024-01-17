// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"

	"cogentcore.org/core/goki/config"
	"cogentcore.org/core/xe"
)

// VersionRelease calls update-version and then release. It is the standard release path.
func VersionRelease(c *config.Config) error { //gti:add
	err := UpdateVersion(c)
	if err != nil {
		return err
	}
	return Release(c)
}

// Release releases the project as a git tag. It should be called after update-version or similar.
func Release(c *config.Config) error { //gti:add
	if c.Type == config.TypeApp {
		return ReleaseApp(c)
	}
	return ReleaseLibrary(c)
}

// ReleaseApp releases the config app.
func ReleaseApp(c *config.Config) error {
	// TODO: actually implement instead of just calling ReleaseLibrary
	return ReleaseLibrary(c)
}

// ReleaseLibrary releases the config library.
func ReleaseLibrary(c *config.Config) error {
	err := PushGitRelease(c)
	if err != nil {
		return fmt.Errorf("error pushing Git release: %w", err)
	}
	return nil
}

// PushGitRelease commits a release commit using Git,
// adds a version tag, and pushes the code and tags
// based on the given config info.
func PushGitRelease(c *config.Config) error {
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
