// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package elide

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestElide(t *testing.T) {
	s := "string for testing purposes"
	assert.Equal(t, "string…", End(s, 7))
	assert.Equal(t, "str…ses", Middle(s, 7))
	assert.Equal(t, "fives", Middle("fives", 5))
}
