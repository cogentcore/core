// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errors

import "testing"

func TestSetLogHook(t *testing.T) {
	var got []error
	prev := SetLogHook(func(err error) bool {
		got = append(got, err)
		return true
	})
	err := New("test error")
	Log(err)
	Log1(1, err)
	Log2(1, 2, err)
	Log(nil)
	if len(got) != 3 {
		t.Errorf("hook got %d errors, want 3", len(got))
	}
	if p := SetLogHook(prev); p == nil {
		t.Error("SetLogHook did not return the hook that it replaces")
	}
}
