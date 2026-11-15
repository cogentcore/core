// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
)

// TestBrowser tests the font browser of the embedded Noto Sans font.
func TestBrowser(t *testing.T) {
	b := core.NewBody()
	fb := makeBody(b)
	assert.NotNil(t, fb.Font)
	core.AssertRender(t, b, "browser")
}
