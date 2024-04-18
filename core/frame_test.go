// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "testing"

func TestFrame(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	NewButton(fr).SetText("First")
	NewButton(fr).SetText("Second")
	NewButton(fr).SetText("Third")
	b.AssertRender(t, "frame/basic")
}
