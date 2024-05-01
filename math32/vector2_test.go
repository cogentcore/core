// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/image/math/fixed"
)

func TestVector2(t *testing.T) {
	assert.Equal(t, Vector2{5, 10}, Vec2(5, 10))
	assert.Equal(t, Vector2{20, 20}, Vector2Scalar(20))
	assert.Equal(t, Vector2{15, -5}, Vector2FromPoint(image.Pt(15, -5)))
	assert.Equal(t, Vector2{8, 3}, Vector2FromFixed(fixed.P(8, 3)))

	v := Vector2{}
	v.Set(-1, 7)
	assert.Equal(t, Vector2{-1, 7}, v)

	v.SetScalar(8.12)
	assert.Equal(t, Vector2{8.12, 8.12}, v)

	v.SetFromVector2i(Vector2i{8, 9})
	assert.Equal(t, Vector2{8, 9}, v)
}
