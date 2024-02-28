// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cie

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSRGB(t *testing.T) {
	assert.Equal(t, float32(0.00015479876), SRGBToLinearComp(0.002))
	assert.Equal(t, float32(0.23302202), SRGBToLinearComp(0.52))
}
