// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"testing"
)

func TestDiff(t *testing.T) {
	astr := []string{"foo", "bar", "baz"}
	bstr := []string{"foo", "bar1", "asdf", "baz"}

	dfs := DiffLines(astr, bstr)
	// fmt.Printf("diffs: %v", dfs)
	if len(dfs) != 3 {
		t.Errorf("diffs != 3")
	}

	pt := dfs.ToPatch(bstr)
	nb := pt.Apply(astr)

	bln := pt.NumBlines()
	// fmt.Printf("num blines: %d\n", bln)
	if bln != 2 {
		t.Errorf("num blines != 2\n")
	}

	// for _, pr := range pt {
	// 	fmt.Printf("%v  blines: %v\n", pr.Op, pr.Blines)
	// }

	// fmt.Printf("nb: %v\n", nb)
	for i := range nb {
		if nb[i] != bstr[i] {
			t.Errorf("Line: %d\t orig: %s  !=  %s", i, bstr[i], nb[i])
		}
	}

	rd := dfs.Reverse()

	// fmt.Printf("rev diffs: %v\n", rd)

	ptr := rd.ToPatch(astr)
	na := ptr.Apply(bstr)

	// fmt.Printf("na: %v\n", na)
	for i := range na {
		if na[i] != astr[i] {
			t.Errorf("Line: %d\t orig: %s  !=  %s", i, astr[i], na[i])
		}
	}

}
