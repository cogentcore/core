// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build goscript

package main

import "testing"

// TestComputeGoScript runs the compute shader in the browser,
// through the JavaScript compiled by goscript.
func TestComputeGoScript(t *testing.T) {
	n := 200_000
	nerr := compute(n)
	if nerr != 0 {
		t.Errorf("%d items out of %d do not match", nerr, n)
	}
}
