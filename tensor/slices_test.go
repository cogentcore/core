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

func TestSlicedExpr(t *testing.T) {
	ft := NewFloat64(3, 4)
	for y := range 3 {
		for x := range 4 {
			v := y*10 + x
			ft.SetFloat(float64(v), y, x)
		}
	}

	res := `[3, 4]
[0]:       0       1       2       3 
[1]:      10      11      12      13 
[2]:      20      21      22      23 
`
	assert.Equal(t, res, ft.String())
	// fmt.Println(ft)

	res = `[1, 1]
[0]:      12 
`
	sl := NewSliced(ft, 1, 2)
	// fmt.Println(sl)
	assert.Equal(t, res, sl.String())

	res = `[1, 4]
[0]:      10      11      12      13 
`
	sl = NewSliced(ft, 1)
	assert.Equal(t, res, sl.String())

	res = `[3, 1]
[0]:       2 
[1]:      12 
[2]:      22 
`
	sl = NewSliced(ft, Elipses, 2)
	assert.Equal(t, res, sl.String())

	res = `[3, 4]
[0]:       3       2       1       0 
[1]:      13      12      11      10 
[2]:      23      22      21      20 
`
	sl = NewSliced(ft, Elipses, Slice{Step: -1})
	assert.Equal(t, res, sl.String())
}
