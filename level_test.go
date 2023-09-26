// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grog

import (
	"log/slog"
	"testing"
)

func TestLevelFromFlags(t *testing.T) {
	l := LevelFromFlags(true, false, false)
	if l != slog.LevelDebug {
		t.Errorf("expected LevelFromFlags(true, false, false) = %v, but got %v", slog.LevelDebug, l)
	}
	l = LevelFromFlags(false, true, true)
	if l != slog.LevelInfo {
		t.Errorf("expected LevelFromFlags(false, true, true) = %v, but got %v", slog.LevelInfo, l)
	}
	l = LevelFromFlags(false, false, true)
	if l != slog.LevelError {
		t.Errorf("expected LevelFromFlags(false, false, true) = %v, but got %v", slog.LevelError, l)
	}
	l = LevelFromFlags(false, false, false)
	if l != slog.LevelWarn {
		t.Errorf("expected LevelFromFlags(false, false, false) = %v, but got %v", slog.LevelWarn, l)
	}
}
