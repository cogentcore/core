// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errors

import (
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustIgnore(t *testing.T) {
	bad := New("bad")
	assert.NotPanics(t, func() { Must(nil) })
	assert.PanicsWithValue(t, bad, func() { Must(bad) })
	assert.Equal(t, 3, Must1(3, nil))
	assert.Panics(t, func() { Must1(3, bad) })
	a, b := Must2(1, "x", nil)
	assert.Equal(t, 1, a)
	assert.Equal(t, "x", b)
	assert.Panics(t, func() { Must2(1, "x", bad) })
	assert.Equal(t, 4, Ignore1(4, bad))
	c, d := Ignore2(5, "y", bad)
	assert.Equal(t, 5, c)
	assert.Equal(t, "y", d)
}

func TestStdlib(t *testing.T) {
	wrapped := fmt.Errorf("open: %w", fs.ErrNotExist)
	assert.True(t, Is(wrapped, fs.ErrNotExist))
	assert.Equal(t, fs.ErrNotExist, Unwrap(wrapped))
	var pe *fs.PathError
	assert.False(t, As(wrapped, &pe))
	assert.True(t, As(&fs.PathError{Op: "open", Err: fs.ErrNotExist}, &pe))
	assert.Equal(t, "open", pe.Op)
	assert.Nil(t, Join(nil, nil))
	j := Join(New("a"), nil, New("b"))
	assert.Equal(t, "a\nb", j.Error())
}
