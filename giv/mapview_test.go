// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"strconv"
	"testing"

	"cogentcore.org/core/core"
)

func TestMapView(t *testing.T) {
	b := core.NewBody()
	m := map[string]bool{}
	for i := 0; i < 10; i++ {
		m["Gopher "+strconv.Itoa(i)] = i%3 == 0
	}
	NewMapView(b).SetMap(&m)
	b.AssertRender(t, "mapview/basic")
}
