// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlice(t *testing.T) {
	assert.Equal(t, 3, Slice{}.Len(3))
	assert.Equal(t, 3, Slice{0, 3, 0}.Len(3))
	assert.Equal(t, 3, Slice{0, 3, 1}.Len(3))

	assert.Equal(t, 2, Slice{0, 0, 2}.Len(3))
	assert.Equal(t, 2, Slice{0, 0, 2}.Len(4))
	assert.Equal(t, 1, Slice{0, 0, 3}.Len(3))
	assert.Equal(t, 2, Slice{0, 0, 3}.Len(4))
	assert.Equal(t, 2, Slice{0, 0, 3}.Len(6))
	assert.Equal(t, 3, Slice{0, 0, 3}.Len(7))

	assert.Equal(t, 1, Slice{-1, 0, 0}.Len(3))
	assert.Equal(t, 2, Slice{0, -1, 0}.Len(3))
	assert.Equal(t, 3, Slice{0, 0, -1}.Len(3))
	assert.Equal(t, 3, Slice{-1, 0, -1}.Len(3))
	assert.Equal(t, 1, Slice{-1, -2, -1}.Len(3))
	assert.Equal(t, 2, Slice{-1, -3, -1}.Len(3))

	assert.Equal(t, 2, Slice{0, 0, -2}.Len(3))
	assert.Equal(t, 2, Slice{0, 0, -2}.Len(4))
	assert.Equal(t, 1, Slice{0, 0, -3}.Len(3))
	assert.Equal(t, 2, Slice{0, 0, -3}.Len(4))
	assert.Equal(t, 2, Slice{0, 0, -3}.Len(6))
	assert.Equal(t, 3, Slice{0, 0, -3}.Len(7))

	assert.Equal(t, []int{0, 1, 2}, Slice{}.IntSlice(3))
	assert.Equal(t, []int{0, 1, 2}, Slice{0, 3, 0}.IntSlice(3))
	assert.Equal(t, []int{0, 1, 2}, Slice{0, 3, 1}.IntSlice(3))

	assert.Equal(t, []int{0, 2}, Slice{0, 0, 2}.IntSlice(3))
	assert.Equal(t, []int{0, 2}, Slice{0, 0, 2}.IntSlice(4))
	assert.Equal(t, []int{0}, Slice{0, 0, 3}.IntSlice(3))
	assert.Equal(t, []int{0, 3}, Slice{0, 0, 3}.IntSlice(4))
	assert.Equal(t, []int{0, 3}, Slice{0, 0, 3}.IntSlice(6))
	assert.Equal(t, []int{0, 3, 6}, Slice{0, 0, 3}.IntSlice(7))

	assert.Equal(t, []int{2}, Slice{-1, 0, 0}.IntSlice(3))
	assert.Equal(t, []int{0, 1}, Slice{0, -1, 0}.IntSlice(3))
	assert.Equal(t, []int{2, 1, 0}, Slice{0, 0, -1}.IntSlice(3))
	assert.Equal(t, []int{2, 1, 0}, Slice{-1, 0, -1}.IntSlice(3))
	assert.Equal(t, []int{2}, Slice{-1, -2, -1}.IntSlice(3))
	assert.Equal(t, []int{2, 1}, Slice{-1, -3, -1}.IntSlice(3))

	assert.Equal(t, []int{2, 0}, Slice{0, 0, -2}.IntSlice(3))
	assert.Equal(t, []int{3, 1}, Slice{0, 0, -2}.IntSlice(4))
	assert.Equal(t, []int{2}, Slice{0, 0, -3}.IntSlice(3))
	assert.Equal(t, []int{3, 0}, Slice{0, 0, -3}.IntSlice(4))
	assert.Equal(t, []int{5, 2}, Slice{0, 0, -3}.IntSlice(6))
	assert.Equal(t, []int{6, 3, 0}, Slice{0, 0, -3}.IntSlice(7))
}
