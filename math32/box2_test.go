// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBox2MulMatrix2(t *testing.T) {
	b := B2(1, 2, 3, 4)
	m := Matrix2{
		1, 2,
		3, 4,
		5, 6,
	}
	expected := B2(12, 16, 20, 28)

	result := b.MulMatrix2(m)

	assert.Equal(t, expected, result)
}
