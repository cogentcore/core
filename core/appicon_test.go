// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/icons"
	"github.com/stretchr/testify/assert"
)

// TestAppIconImages tests that the large image of the app icon, for the
// Dock on macOS, is only for the app's own icon, not the default one.
func TestAppIconImages(t *testing.T) {
	icon := AppIcon
	t.Cleanup(func() {
		AppIcon = icon
		appIconImagesCache = nil
	})
	sizes := func() []int {
		appIconImagesCache = nil
		var szs []int
		for _, img := range appIconImages() {
			szs = append(szs, img.Bounds().Dx())
		}
		return szs
	}
	AppIcon = string(icons.CogentCore)
	assert.Equal(t, []int{16, 32, 48}, sizes(), "the default icon")
	AppIcon = string(icons.Folder)
	assert.Equal(t, []int{16, 32, 48, 512}, sizes(), "the app's own icon")
}
