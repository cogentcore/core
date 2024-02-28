// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cam02

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLuminanceAdapt(t *testing.T) {
	assert.Equal(t, float32(1), LuminanceAdapt(200))
}
