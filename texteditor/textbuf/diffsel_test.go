// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"fmt"
	"testing"
)

func TestDiffSel(t *testing.T) {
	astr := []string{
		"a one",
		"a two",
		"a three",
		"a four",
	}
	bstr := []string{
		"a two",
		"b three",
		"a four",
		"b five",
	}

	ds := NewDiffSelected(astr, bstr)
	// fmt.Println(ds.Diffs)

	ds.AtoB(0)
	ds.AtoB(1)
	ds.AtoB(2)
	ds.AtoB(3)
	ds.AtoB(4)

	// fmt.Println(strings.Join(ds.B.Edit, "\n"))
	for i := range astr {
		if astr[i] != ds.B.Edit[i] {
			t.Error(fmt.Sprintln(i, astr[i], "!=", ds.B.Edit[i]))
		}
	}

	// fmt.Println("## undo:")
	ds.Undo()
	ds.Undo()
	ds.Undo()
	ds.Undo()
	ds.Undo()
	// fmt.Println(strings.Join(ds.B.Edit, "\n"))
	for i := range bstr {
		if bstr[i] != ds.B.Edit[i] {
			t.Error(fmt.Sprintln(i, bstr[i], "!=", ds.B.Edit[i]))
		}
	}

	// opposite order
	ds.AtoB(4)
	ds.AtoB(3)
	ds.AtoB(2)
	ds.AtoB(1)
	ds.AtoB(0)

	// fmt.Println(strings.Join(ds.B.Edit, "\n"))
	for i := range astr {
		if astr[i] != ds.B.Edit[i] {
			t.Error(fmt.Sprintln(i, astr[i], "!=", ds.B.Edit[i]))
		}
	}

	ds.BtoA(0)
	ds.BtoA(1)
	ds.BtoA(2)
	ds.BtoA(3)
	ds.BtoA(4)

	// fmt.Println(strings.Join(ds.A.Edit, "\n"))
	for i := range bstr {
		if bstr[i] != ds.A.Edit[i] {
			t.Error(fmt.Sprintln(i, bstr[i], "!=", ds.A.Edit[i]))
		}
	}

	ds.Undo()
	ds.Undo()
	ds.Undo()
	ds.Undo()
	ds.Undo()
	// fmt.Println(strings.Join(ds.B.Edit, "\n"))
	for i := range bstr {
		if astr[i] != ds.A.Edit[i] {
			t.Error(fmt.Sprintln(i, astr[i], "!=", ds.A.Edit[i]))
		}
	}

	ds.BtoA(4)
	ds.BtoA(3)
	ds.BtoA(2)
	ds.BtoA(1)
	ds.BtoA(0)

	// fmt.Println(strings.Join(ds.A.Edit, "\n"))
	for i := range bstr {
		if bstr[i] != ds.A.Edit[i] {
			t.Error(fmt.Sprintln(i, bstr[i], "!=", ds.A.Edit[i]))
		}
	}
}
