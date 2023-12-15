// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dedupe

import (
	"testing"
)

func TestDeDupe(t *testing.T) {
	str := []string{"apple", "banana", "apple", "banana"}

	dds := DeDupe(str)
	// fmt.Println(dd)
	if len(dds) != 2 {
		t.Error("len != 2")
	}

	ints := []int{1, 2, 2, 3, 3, 3, 1, 1, 4, 2, 3, 1}

	ddi := DeDupe(ints)
	// fmt.Println(ddi)
	if len(ddi) != 4 {
		t.Error("len != 4")
	}
}
