// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cam16

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlend(t *testing.T) {
	c := Blend(50, color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255})
	assert.Equal(t, color.RGBA{112, 112, 111, 255}, c)
}
