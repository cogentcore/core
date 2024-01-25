// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"

	"cogentcore.org/core/goosi"
)

func TestSnackbar(t *testing.T) {
	b := NewBody()
	NewLabel(b).SetText("Hello")
	MessageSnackbar(b, "Test")
	b.NewWindow().Run()
	goosi.AssertCapture(t, filepath.Join("snackbar", "basic"))
}
