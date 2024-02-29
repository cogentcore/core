// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tolassert

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
