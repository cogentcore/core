// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"errors"
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
	times := []time.Duration{0, 25 * time.Millisecond, 75 * time.Millisecond}
	for _, tm := range times {
		tm := tm
		b := NewBody()
		b.Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(300))
		})
		NewLabel(b).SetText(tm.String())

		b.AssertScreenRender(t, filepath.Join("snackbar", tm.String()), func() {
			MessageSnackbar(b, "Test")
			time.Sleep(tm)
		})
	}

	// test making two
	for _, tm := range times {
		tm := tm
		b := NewBody()
		b.Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(300))
		})
		NewLabel(b).SetText(tm.String() + "-two")

		b.AssertScreenRender(t, filepath.Join("snackbar", tm.String()+"-two"), func() {
			MessageSnackbar(b, "Test One")
			time.Sleep(tm / 2)
			MessageSnackbar(b, "Test Two")
			time.Sleep(tm)
		})
	}
}

func TestErrorSnackbar(t *testing.T) {
	nb := func() *Body {
		b := NewBody()
		b.Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(300))
		})
		return b
	}

	b := nb()
	b.AssertScreenRender(t, filepath.Join("snackbar", "no-error"), func() {
		ErrorSnackbar(b, nil)
	})

	b = nb()
	b.AssertScreenRender(t, filepath.Join("snackbar", "error"), func() {
		ErrorSnackbar(b, errors.New("file not found"))
	})

	b = nb()
	b.AssertScreenRender(t, filepath.Join("snackbar", "error-label"), func() {
		ErrorSnackbar(b, errors.New("file not found"), "Error loading page")
	})
}
