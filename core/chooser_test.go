// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "testing"

func TestChooserSetStrings(t *testing.T) {
	b := NewBody()
	NewChooser(b).SetStrings("macOS", "Windows", "Linux")
	b.AssertRender(t, "chooser/set-strings")
}
