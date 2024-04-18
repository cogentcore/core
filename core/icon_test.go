// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/icons"
)

func TestIcon(t *testing.T) {
	b := NewBody()
	NewIcon(b).SetIcon(icons.Home)
	b.AssertRender(t, "icon/basic")
}

func TestIconFilled(t *testing.T) {
	b := NewBody()
	NewIcon(b).SetIcon(icons.Home.Fill())
	b.AssertRender(t, "icon/filled")
}
