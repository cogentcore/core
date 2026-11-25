// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"testing"

	"cogentcore.org/core/base/goscript"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	goscript.Skip(t, "running external commands is not supported")
	assert.NoError(t, Run("go", "version"))
	assert.NoError(t, Run("git", "version"))
	assert.NoError(t, Run("echo", " hello"))
}

func TestError(t *testing.T) {
	assert.Error(t, Run("go", "bild"))
}

// TestNoExpand tests that $FOO references in args are expanded, unless
// [Config.NoExpand], when they are passed as is.
func TestNoExpand(t *testing.T) {
	goscript.Skip(t, "running external commands is not supported")
	c := Major().SetBuffer(false)
	c.Env = map[string]string{"FOO": "foo"}
	out, err := c.Output("echo", "$FOO", "a$#b")
	assert.NoError(t, err)
	assert.Equal(t, "foo ab", out)
	c.NoExpand = true
	out, err = c.Output("echo", "$FOO", "a$#b")
	assert.NoError(t, err)
	assert.Equal(t, "$FOO a$#b", out)
}
