// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTextInputBefore(t *testing.T) {
	assert.Equal(t, "", TextInputBefore(nil))
	assert.Equal(t, "Milk and", TextInputBefore([]rune("Milk and")))
	assert.Equal(t, "second line", TextInputBefore([]rune("first\nsecond line")), "after a newline")
	long := strings.Repeat("ab", TextInputBeforeMax)
	assert.Equal(t, long[len(long)-TextInputBeforeMax:], TextInputBefore([]rune(long)), "at most the max")
}
