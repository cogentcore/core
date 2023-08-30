// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"fmt"
	"os/exec"

	"goki.dev/goki/config"
)

// Release releases the config project
// by calling [ReleaseApp] if it is an app
// and [ReleaseLibrary] if it is a library.
func Release(c *config.Config) error {
	if c.Type == config.TypeApp {
		return ReleaseApp(c)
	}
	return ReleaseLibrary(c)
}

// ReleaseApp releases the config app.
func ReleaseApp(c *config.Config) error {
	// TODO: implement
	return nil
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
	tc := exec.Command("git", "tag", "-a", c.Version, "-m", c.Version+" release")
	_, err := RunCmd(tc)
	if err != nil {
		return fmt.Errorf("error tagging release: %w", err)
	}

	ptc := exec.Command("git", "push", "origin", "--tags")
	_, err = RunCmd(ptc)
	if err != nil {
		return fmt.Errorf("error pushing tags: %w", err)
	}

	return nil
}
