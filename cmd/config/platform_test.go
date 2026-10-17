// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlatformSetString(t *testing.T) {
	var p Platform
	assert.NoError(t, p.SetString("wasm"))
	assert.Equal(t, Platform{OS: "wasm", Arch: "*"}, p)

	// The old name of the wasm platform still works.
	assert.NoError(t, p.SetString("web"))
	assert.Equal(t, "wasm", p.OS)

	assert.NoError(t, p.SetString("js"))
	assert.Equal(t, "js", p.OS)
	assert.Error(t, p.SetString("plan9"))
}
