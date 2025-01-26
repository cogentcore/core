// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows && registry-test

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistryWindows(t *testing.T) {
	assert.NoError(t, windowsRegistryAddPath(`C:\w64devkit\bin`))
}
