// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grog

import "testing"

func TestLevelFromFlags(t *testing.T) {
	l := LevelFromFlags(true, false, false)
	if l != Debug {
		t.Errorf("expected LevelFromFlags(true, false, false) = %v, but got %v", Debug, l)
	}
	l = LevelFromFlags(false, true, true)
	if l != Info {
		t.Errorf("expected LevelFromFlags(false, true, true) = %v, but got %v", Info, l)
	}
	l = LevelFromFlags(false, false, true)
	if l != Error {
		t.Errorf("expected LevelFromFlags(false, false, true) = %v, but got %v", Error, l)
	}
	l = LevelFromFlags(false, false, false)
	if l != Warn {
		t.Errorf("expected LevelFromFlags(false, false, false) = %v, but got %v", Warn, l)
	}
}
