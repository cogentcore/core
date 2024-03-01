// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"testing"

	"cogentcore.org/core/gi"
)

func TestStructViewInline(t *testing.T) {
	b := gi.NewBody()
	NewStructViewInline(b).SetStruct(myStructValue)
	b.AssertRender(t, "structviewinline/basic")
}
