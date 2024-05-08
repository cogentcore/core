// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package atomiccounter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounter(t *testing.T) {
	var c Counter
	assert.Equal(t, int64(0), c.Value())
	assert.Equal(t, int64(1), c.Inc())
	assert.Equal(t, int64(1), c.Value())
	assert.Equal(t, int64(2), c.Add(1))
	assert.Equal(t, int64(2), c.Value())
	assert.Equal(t, int64(1), c.Sub(1))
	assert.Equal(t, int64(1), c.Value())
	assert.Equal(t, int64(0), c.Dec())
	assert.Equal(t, int64(0), c.Value())
	assert.Equal(t, int64(0), c.Swap(1))
	assert.Equal(t, int64(1), c.Value())
	c.Set(2)
	assert.Equal(t, int64(2), c.Value())
}
