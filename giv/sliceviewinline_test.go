// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"testing"

	"cogentcore.org/core/gi"
)

func TestSliceViewInline(t *testing.T) {
	b := gi.NewBody()
	sl := make([]float32, 10)
	for i := range sl {
		fi := float32(i)
		sl[i] = 2*fi + (8-fi)/10
	}
	NewSliceViewInline(b).SetSlice(&sl)
	b.AssertRender(t, "sliceviewinline/basic")
}
