// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package minmax

import "testing"

var testNums = []struct {
	in, out float64
	below   bool
}{
	{1.2, 2, false},
	{-1.2, -1, false},
	{2.2, 5, false},
	{4.9, 5, false},
	{5.001, 10, false},
	{1200, 2000, false},
	{-1200, -1000, false},
	{2.2e-10, 5e-10, false},
	{4.9e10, 5e10, false},
	{5.001e6, 1.0e7, false},
	{1.2, 1, true},
	{2.2, 2, true},
	{5.001, 5, true},
	{8.001, 5, true},
	{10.00, 10, true},
	{10.01, 10, true},
	{11.01, 10, true},
	{-1.2, -2, true},
	{-.57, -1, true},
	{-2.57, -5, true},
	{-5.57, -10, true},
}

func TestNiceRoundNum(t *testing.T) {
	for _, tst := range testNums {
		nn := NiceRoundNumber(tst.in, tst.below)
		if nn != tst.out {
			t.Errorf("tst case: %v failed: nn = %v\n", tst, nn)
		}
	}
}
