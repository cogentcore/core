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
	"cogentcore.org/core/units"
)

func newBodyForSnackbar() *Body {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Dp(300))
	})
	return b
}

func TestSnackbarCustom(t *testing.T) {
	b := newBodyForSnackbar()
	b.AssertRenderScreen(t, filepath.Join("snackbar", "text"), func() {
		NewBody().AddSnackbarText("Files updated").NewSnackbar(b).Run()
	})

	b = newBodyForSnackbar()
	b.AssertRenderScreen(t, filepath.Join("snackbar", "button"), func() {
		NewBody().AddSnackbarText("Files updated").AddSnackbarButton("Refresh").NewSnackbar(b).Run()
	})

	b = newBodyForSnackbar()
	b.AssertRenderScreen(t, filepath.Join("snackbar", "icon"), func() {
		NewBody().AddSnackbarText("Files updated").AddSnackbarIcon(icons.Close).NewSnackbar(b).Run()
	})

	b = newBodyForSnackbar()
	b.AssertRenderScreen(t, filepath.Join("snackbar", "button-icon"), func() {
		NewBody().AddSnackbarText("Files updated").AddSnackbarButton("Refresh").AddSnackbarIcon(icons.Close).NewSnackbar(b).Run()
	})
}

func TestSnackbarMessage(t *testing.T) {
	b := newBodyForSnackbar()
	b.AssertRenderScreen(t, filepath.Join("snackbar", "message"), func() {
		MessageSnackbar(b, "New messages loaded")
	})
}

func TestSnackbarError(t *testing.T) {
	b := newBodyForSnackbar()
	b.AssertRenderScreen(t, filepath.Join("snackbar", "no-error"), func() {
		ErrorSnackbar(b, nil)
	})

	b = newBodyForSnackbar()
	b.AssertRenderScreen(t, filepath.Join("snackbar", "error"), func() {
		ErrorSnackbar(b, errors.New("file not found"))
	})

	b = newBodyForSnackbar()
	b.AssertRenderScreen(t, filepath.Join("snackbar", "error-label"), func() {
		ErrorSnackbar(b, errors.New("file not found"), "Error loading page")
	})
}

func TestSnackbarTime(t *testing.T) {
	ptimeout := SystemSettings.SnackbarTimeout
	SystemSettings.SnackbarTimeout = 50 * time.Millisecond
	defer func() {
		SystemSettings.SnackbarTimeout = ptimeout
	}()
	times := []time.Duration{0, 25 * time.Millisecond, 75 * time.Millisecond}
	for _, tm := range times {
		b := NewBody()
		b.Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(300))
		})
		NewLabel(b).SetText(tm.String())

		b.AssertRenderScreen(t, filepath.Join("snackbar", tm.String()), func() {
			MessageSnackbar(b, "Test")
			time.Sleep(tm)
		})
	}

	// test making two
	for _, tm := range times {
		b := newBodyForSnackbar()
		NewLabel(b).SetText(tm.String() + "-two")

		b.AssertRenderScreen(t, filepath.Join("snackbar", tm.String()+"-two"), func() {
			MessageSnackbar(b, "Test One")
			time.Sleep(tm / 2)
			MessageSnackbar(b, "Test Two")
			time.Sleep(tm)
		})
	}
}
