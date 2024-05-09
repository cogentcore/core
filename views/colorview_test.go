// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
)

func TestColorView(t *testing.T) {
	b := core.NewBody()
	NewColorView(b).SetColor(colors.Orange)
	b.AssertRender(t, "color-view/basic")
}
