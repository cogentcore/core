// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package num

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBool(t *testing.T) {
	assert.True(t, ToBool(1))
	assert.False(t, ToBool(0.0))

	f32 := FromBool[float32](true)
	assert.Equal(t, float32(1), f32)

	SetFromBool(&f32, false)
	assert.Equal(t, float32(0), f32)
}

func TestAbs(t *testing.T) {
	assert.Equal(t, 22, Abs(-22))

	// This correctly does not compile:
	// assert.Equal(t, uint8(5), Abs(uint8(5)))

	assert.Equal(t, 4.31, Abs(-4.31))
}
