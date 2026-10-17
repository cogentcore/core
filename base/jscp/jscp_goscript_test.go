// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js && goscript

package jscp

import "testing"

func TestBytesToJS(t *testing.T) {
	buf := make([]byte, 16, 32)
	for i := range buf {
		buf[i] = byte(i)
	}
	b := buf[3:7]
	v := BytesToJS(b)
	if n := v.Get("length").Int(); n != 4 {
		t.Fatalf("length = %d, want 4", n)
	}
	for i := range b {
		if got := v.Index(i).Int(); got != int(b[i]) {
			t.Errorf("index %d = %d, want %d", i, got, b[i])
		}
	}

	// The result is a view of the same memory, not a copy.
	b[1] = 200
	if got := v.Index(1).Int(); got != 200 {
		t.Errorf("after Go write, index 1 = %d, want 200", got)
	}
	v.SetIndex(2, 77)
	if b[2] != 77 {
		t.Errorf("after JS write, b[2] = %d, want 77", b[2])
	}
}
