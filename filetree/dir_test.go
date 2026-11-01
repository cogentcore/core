// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirFlagMapCopy(t *testing.T) {
	// copying an empty map leaves the map usable (it used to stay locked)
	var dm, empty DirFlagMap
	dm.Copy(&empty)
	dm.setOpen("a", true)
	var cp DirFlagMap
	cp.Copy(&dm)
	assert.True(t, cp.isOpen("a"))
}
