// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"os/user"
	"path/filepath"
	"testing"

	"cogentcore.org/core/system"
	"github.com/stretchr/testify/assert"
)

// displayAvailable returns whether there is a display, which there always
// is on macOS (except over ssh: set CORE_NO_GUI_TESTS there).
func displayAvailable() bool { return true }

func TestPlatformDarwin(t *testing.T) {
	assert.Equal(t, system.MacOS, TheApp.Platform())
	usr, err := user.Current()
	if assert.NoError(t, err) {
		assert.Equal(t, filepath.Join(usr.HomeDir, "Library"), TheApp.DataDir())
	}
}
