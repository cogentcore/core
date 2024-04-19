// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"testing"

	"cogentcore.org/core/core"
)

func TestTree(t *testing.T) {
	b := core.NewBody()
	NewTree(b).OpenPath("../events")
	b.AssertRender(t, "basic")
}
