// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"testing"

	"cogentcore.org/core/core"
)

func TestSliceView(t *testing.T) {
	b := core.NewBody()
	sl := make([]float32, 10)
	for i := range sl {
		fi := float32(i)
		sl[i] = 2*fi + (8-fi)/10
	}
	NewSliceView(b).SetSlice(&sl)
	b.AssertRender(t, "sliceview/basic")
}
