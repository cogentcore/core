// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tolassert

//core:js-test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockTestingT struct{}

func (t *mockTestingT) Errorf(format string, args ...any) {}

var testingT = &mockTestingT{}

func TestEqual(t *testing.T) {
	assert.True(t, Equal(testingT, 132, 132.0004))
	assert.False(t, Equal(testingT, 132, 132.004))
}

func TestEqualTol(t *testing.T) {
	assert.True(t, EqualTol(testingT, 132.0, 136, 4))
	assert.False(t, EqualTol(testingT, 132.0, 136, 3))
}

func TestEqualTolSlice(t *testing.T) {
	assert.True(t, EqualTolSlice(testingT, []float64{1, 2}, []float64{1.05, 2}, 0.1), "within the tolerance is true")
	assert.False(t, EqualTolSlice(testingT, []float64{1, 2}, []float64{1, 3}, 0.1))
	assert.False(t, EqualTolSlice(testingT, []float32{1, 2}, []float32{1}, 0.1), "a shorter actual fails, without panicking")
	assert.False(t, EqualTolSlice(testingT, []float32{1}, []float32{1, 2}, 0.1), "a longer actual fails")
	assert.True(t, EqualTolSlice(testingT, nil, nil, 0.1))
}
