// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/colors"
)

func TestColorPicker(t *testing.T) {
	b := NewBody()
	NewColorPicker(b).SetColor(colors.Orange)
	b.AssertRender(t, "color-picker/basic")
}
