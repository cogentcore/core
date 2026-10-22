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

	// The web platform of earlier versions is an error that names js and wasm.
	err := p.SetString("web")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "use js")
		assert.Contains(t, err.Error(), "wasm")
	}

	assert.NoError(t, p.SetString("js"))
	assert.Equal(t, "js", p.OS)
	assert.Error(t, p.SetString("plan9"))
}
