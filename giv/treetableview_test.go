// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"testing"

	"cogentcore.org/core/gi"
)

func TestTreeTableView(t *testing.T) {
	b := gi.NewBody()
	NewTreeTableView(b)
	b.AssertRender(t, "treetable/basic")
}
