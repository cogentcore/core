// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"
)

func TestTabs(t *testing.T) {
	for _, typ := range TabTypesValues() {
		sc := NewScene()
		ts := NewTabs(sc).SetType(typ)
		ts.NewTab("Search")
		ts.NewTab("Discover")
		ts.NewTab("History")
		sc.AssertPixelsOnShow(t, testName("tabs", typ))
	}
}
