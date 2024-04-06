// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"testing"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

func TestCallFunc(t *testing.T) {
	called := false
	myFunc := func() {
		called = true
	}
	CallFunc(tree.NewRoot[*gi.Frame](), myFunc)
	assert.True(t, called)
}
