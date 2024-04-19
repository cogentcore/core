// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"testing"

	"cogentcore.org/core/core"
)

func TestEditor(t *testing.T) {
	b := core.NewBody()
	NewSoloEditor(b).Buffer.SetTextString("Hello, world!")
	b.AssertRender(t, "basic")
}
