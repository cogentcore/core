// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWeightFromFloat32 tests the weights of numerical weights.
func TestWeightFromFloat32(t *testing.T) {
	assert.Equal(t, Thin, WeightFromFloat32(100))
	assert.Equal(t, Normal, WeightFromFloat32(400))
	assert.Equal(t, Bold, WeightFromFloat32(700))
	assert.Equal(t, Semibold, WeightFromFloat32(620))
	assert.Equal(t, Black, WeightFromFloat32(1000))
	assert.Equal(t, Thin, WeightFromFloat32(0))
	for _, w := range WeightsValues() {
		assert.Equal(t, w, WeightFromFloat32(w.ToFloat32()))
	}
}
