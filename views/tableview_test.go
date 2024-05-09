// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/core"
)

type language struct {
	Name   string
	Rating int
}

func TestTableView(t *testing.T) {
	b := core.NewBody()
	NewTableView(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
	b.AssertRender(t, "table-view/basic")
}

func TestTableViewReadOnly(t *testing.T) {
	b := core.NewBody()
	NewTableView(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}}).SetReadOnly(true)
	b.AssertRender(t, "table-view/read-only")
}
