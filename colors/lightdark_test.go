// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLightDark tests that light-dark() is its light color (as in the
// SVGs of MDN), which was taken as a function like lighten-10.
func TestLightDark(t *testing.T) {
	c, err := FromString("light-dark(#0a5dbd,#a4cefe)")
	require.NoError(t, err)
	assert.Equal(t, color.RGBA{0x0a, 0x5d, 0xbd, 0xff}, c)
	c, err = FromString("light-dark(rgb(1, 2, 3), white)")
	require.NoError(t, err)
	assert.Equal(t, color.RGBA{1, 2, 3, 0xff}, c)
	c, err = FromString("lighten-10", Black)
	require.NoError(t, err)
	assert.NotEqual(t, color.RGBA{0, 0, 0, 0xff}, c, "lighten still works")
}
