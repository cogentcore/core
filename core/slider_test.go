// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "testing"

func TestSlider(t *testing.T) {
	b := NewBody()
	NewSlider(b)
	b.AssertRender(t, "slider/basic")
}
