// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuatSlerp(t *testing.T) {
	q1 := Quat{X: 1, Y: 2, Z: 3, W: 4}
	q2 := Quat{X: 5, Y: 6, Z: 7, W: 8}

	q := q1
	q.Slerp(q2, 0)
	assert.Equal(t, q1, q)

	q = q1
	q.Slerp(q2, 1)
	assert.Equal(t, q2, q)

	q = q1
	q.Slerp(q2, 0.5)
	assert.Equal(t, Quat{X: 1, Y: 2, Z: 3, W: 4}, q)
}
