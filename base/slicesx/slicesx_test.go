// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slicesx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetLength(t *testing.T) {
	var s []int
	s = SetLength(s, 3)
	assert.Equal(t, 3, len(s))

	s[2] = 2
	s = SetLength(s, 40)
	assert.Equal(t, 40, len(s))
	assert.Equal(t, 2, s[2])

	s = SetLength(s, 4)
	assert.Equal(t, 4, len(s))
	assert.Equal(t, 2, s[2])
}
