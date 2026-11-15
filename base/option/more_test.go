// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package option

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionOps(t *testing.T) {
	o := New(5)
	assert.True(t, o.Valid)
	assert.Equal(t, 5, o.Or(9))
	assert.True(t, o.ShouldSave())
	assert.True(t, o.ShouldDisplay("Value"))
	assert.True(t, o.ShouldDisplay("Valid"))
	o.Clear()
	assert.False(t, o.Valid)
	assert.Equal(t, 9, o.Or(9))
	assert.Equal(t, 5, o.Value, "the value is kept")
	assert.False(t, o.ShouldSave())
	assert.False(t, o.ShouldDisplay("Value"))
	assert.True(t, o.ShouldDisplay("Valid"))
	var z Option[string]
	assert.Equal(t, "def", z.Or("def"))
	assert.Equal(t, "x", z.Set("x").Or("def"))
}
