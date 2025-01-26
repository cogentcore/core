// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

func TestDialogMessage(t *testing.T) {
	t.Skip("TODO(#1456): fix this test")
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(300))
	})
	b.AssertRender(t, "dialog/message", func() {
		MessageDialog(b, "Something happened", "Message")
	})
}
