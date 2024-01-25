// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

func TestSnackbarTime(t *testing.T) {
	SystemSettings.SnackbarTimeout = 50 * time.Millisecond
	defer func() {
		SystemSettings.SnackbarTimeout = 7 * time.Second
	}()
	times := []time.Duration{0, 25 * time.Millisecond, 25 * time.Millisecond}
	for _, tm := range times {
		tm := tm
		b := NewBody()
		b.Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(300))
		})
		b.AssertScreenRender(t, filepath.Join("snackbar", tm.String()), func() {
			MessageSnackbar(b, "Test")
			time.Sleep(tm)
		})
	}
}
