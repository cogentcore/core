// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	assert.NoError(t, Run("go", "version"))
	assert.NoError(t, Run("git", "version"))
	assert.NoError(t, Run("echo", " hello"))
}

func TestError(t *testing.T) {
	assert.Error(t, Run("go", "bild"))
}
