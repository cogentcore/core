// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

func TestSnackbar(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Dp(300))
	})
	b.AssertScreenRender(t, filepath.Join("snackbar", "basic"), func() {
		MessageSnackbar(b, "Test")
	})
}
