// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"testing"

	"cogentcore.org/core/gi"
)

func TestTableView(t *testing.T) {
	b := gi.NewBody()
	sl := make([]myStruct, 10)
	NewTableView(b).SetSlice(&sl)
	b.AssertRender(t, "tableview/basic")
}
