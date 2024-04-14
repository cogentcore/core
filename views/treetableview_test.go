// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/core"
)

func TestTreeTableView(t *testing.T) {
	b := core.NewBody()
	NewTreeTableView(b)
	b.AssertRender(t, "treetable/basic")
}
