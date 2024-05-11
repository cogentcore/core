// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
)

func TestCallFunc(t *testing.T) {
	called := false
	myFunc := func() {
		called = true
	}
	CallFunc(nil, myFunc)
	assert.True(t, called)
}

func TestCallFuncArgs(t *testing.T) {
	b := core.NewBody()
	myFunc := func(a int, b string) {}
	CallFunc(b, myFunc)
	b.AssertRenderScreen(t, "func-button/args")
}
