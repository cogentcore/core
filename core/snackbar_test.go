// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

func newBodyForSnackbar() *Body {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(300))
	})
	return b
}

func TestSnackbarCustom(t *testing.T) {
	b := newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "text"), func() {
		NewBody().AddSnackbarText("Files updated").RunSnackbar(b)
	})

	b = newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "button"), func() {
		NewBody().AddSnackbarText("Files updated").AddSnackbarButton("Refresh").RunSnackbar(b)
	})

	b = newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "icon"), func() {
		NewBody().AddSnackbarText("Files updated").AddSnackbarIcon(icons.Close).RunSnackbar(b)
	})

	b = newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "button-icon"), func() {
		NewBody().AddSnackbarText("Files updated").AddSnackbarButton("Refresh").AddSnackbarIcon(icons.Close).RunSnackbar(b)
	})
}

func TestSnackbarMessage(t *testing.T) {
	b := newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "message"), func() {
		MessageSnackbar(b, "New messages loaded")
	})
}

func TestSnackbarError(t *testing.T) {
	b := newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "no-error"), func() {
		ErrorSnackbar(b, nil)
	})

	b = newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "error"), func() {
		ErrorSnackbar(b, errors.New("file not found"))
	})

	b = newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "error-label"), func() {
		ErrorSnackbar(b, errors.New("file not found"), "Error loading page")
	})
}

func TestSnackbarTime(t *testing.T) {
	t.Skip("TODO(#1456): fix this test")
	ptimeout := SystemSettings.SnackbarTimeout
	SystemSettings.SnackbarTimeout = 50 * time.Millisecond
	defer func() {
		SystemSettings.SnackbarTimeout = ptimeout
	}()
	times := []time.Duration{0, 25 * time.Millisecond, 75 * time.Millisecond}
	for _, tm := range times {
		b := NewBody()
		b.Styler(func(s *styles.Style) {
			s.Min.Set(units.Dp(300))
		})
		NewText(b).SetText(tm.String())

		b.AssertRender(t, filepath.Join("snackbar", tm.String()), func() {
			MessageSnackbar(b, "Test")
			time.Sleep(tm)
		})
	}

	// test making two
	for _, tm := range times {
		b := newBodyForSnackbar()
		NewText(b).SetText(tm.String() + "-two")

		b.AssertRender(t, filepath.Join("snackbar", tm.String()+"-two"), func() {
			MessageSnackbar(b, "Test One")
			time.Sleep(tm / 2)
			MessageSnackbar(b, "Test Two")
			time.Sleep(tm)
		})
	}
}
