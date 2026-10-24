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
		b.AsyncLock()
		defer b.AsyncUnlock()
		NewBody().AddSnackbarText("Files updated").RunSnackbar(b)
	})

	b = newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "button"), func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		NewBody().AddSnackbarText("Files updated").AddSnackbarButton("Refresh").RunSnackbar(b)
	})

	b = newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "icon"), func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		NewBody().AddSnackbarText("Files updated").AddSnackbarIcon(icons.Close).RunSnackbar(b)
	})

	b = newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "button-icon"), func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		NewBody().AddSnackbarText("Files updated").AddSnackbarButton("Refresh").AddSnackbarIcon(icons.Close).RunSnackbar(b)
	})
}

func TestSnackbarMessage(t *testing.T) {
	b := newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "message"), func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		MessageSnackbar(b, "New messages loaded")
	})
}

func TestSnackbarError(t *testing.T) {
	b := newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "no-error"), func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ErrorSnackbar(b, nil)
	})

	b = newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "error"), func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ErrorSnackbar(b, errors.New("file not found"))
	})

	b = newBodyForSnackbar()
	b.AssertRender(t, filepath.Join("snackbar", "error-label"), func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ErrorSnackbar(b, errors.New("file not found"), "Error loading page")
	})
}

// TestSnackbarTime tests that a snackbar is shown until its timeout, after
// which it is removed, and that a new snackbar replaces the one shown.
// A snackbar that is shown has no timeout, and the removal is waited for,
// so that the result does not depend on how fast the test runs.
func TestSnackbarTime(t *testing.T) {
	ptimeout := TimingSettings.SnackbarTimeout
	defer func() {
		TimingSettings.SnackbarTimeout = ptimeout
	}()
	for _, expired := range []bool{false, true} {
		name := "shown"
		TimingSettings.SnackbarTimeout = 0
		if expired {
			name = "expired"
			TimingSettings.SnackbarTimeout = time.Millisecond
		}
		b := newBodyForSnackbar()
		NewText(b).SetText(name)
		b.AssertRender(t, filepath.Join("snackbar", name), func() {
			b.AsyncLock()
			st := NewBody().AddSnackbarText("Test").RunSnackbar(b)
			b.AsyncUnlock()
			if expired {
				waitStageDeleted(b, st)
			}
		})
	}

	TimingSettings.SnackbarTimeout = 0
	b := newBodyForSnackbar()
	NewText(b).SetText("two")
	b.AssertRender(t, filepath.Join("snackbar", "two"), func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		MessageSnackbar(b, "Test One")
		MessageSnackbar(b, "Test Two")
	})
}

// waitStageDeleted waits until the given popup stage in the window of the
// given body has been deleted, as when its timeout has passed.
func waitStageDeleted(b *Body, st *Stage) {
	for {
		b.AsyncLock()
		deleted := st.Main == nil
		b.AsyncUnlock()
		if deleted {
			return
		}
		time.Sleep(time.Millisecond)
	}
}
