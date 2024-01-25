// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"
)

func TestSnackbar(t *testing.T) {
	b := NewBody()
	NewLabel(b).SetText("Hello")
	MessageSnackbar(b, "Test")
	b.AssertScreenRender(t, filepath.Join("snackbar", "basic"))
}
