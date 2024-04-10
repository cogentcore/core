// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/core"
)

func TestTableView(t *testing.T) {
	b := core.NewBody()
	sl := make([]myStruct, 10)
	for i := range sl {
		sl[i] = myStruct{
			Name:    "Person",
			Age:     30 - i,
			Rating:  1.6 * float32(i),
			LikesGo: i%2 == 0,
		}
	}
	NewTableView(b).SetSlice(&sl)
	b.AssertRender(t, "tableview/basic")
}
