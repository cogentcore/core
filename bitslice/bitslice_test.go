// Copyright (c) 2024, The Cogent Core Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitslice

import (
	"testing"
)

func TestBitSlice10(t *testing.T) {
	bs := Make(10, 0)

	ln := bs.Len()
	if ln != 10 {
		t.Errorf("len: %v != 10\n", ln)
	}

	// fmt.Printf("empty: %v\n", bs.String())
	var ex, out string
	ex = "[0 0 0 0 0 0 0 0 0 0 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("empty != %v", out)
	}

	bs.Set(2, true)
	// fmt.Printf("2=true: %v\n", bs.String())
	ex = "[0 0 1 0 0 0 0 0 0 0 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("2=true != %v", out)
	}

	bs.Set(4, true)
	// fmt.Printf("4=true: %v\n", bs.String())
	ex = "[0 0 1 0 1 0 0 0 0 0 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("4=true != %v", out)
	}

	bs.Set(9, true)
	// fmt.Printf("9=true: %v\n", bs.String())
	ex = "[0 0 1 0 1 0 0 0 0 1 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("9=true != %v", out)
	}

	bs.Append(true)
	// fmt.Printf("append true: %v\n", bs.String())
	ex = "[0 0 1 0 1 0 0 0 0 1 1 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("append true != %v", out)
	}

	bs.Append(false)
	// fmt.Printf("append false: %v\n", bs.String())
	ex = "[0 0 1 0 1 0 0 0 0 1 1 0 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("append false != %v", out)
	}

	ss := bs.SubSlice(2, 6)
	// fmt.Printf("subslice: %v\n", ss.String())
	ex = "[1 0 1 0 ]"
	out = ss.String()
	if out != ex {
		t.Errorf("subslice[2,6] != %v", out)
	}

	ds := bs.Delete(2, 4)
	// fmt.Printf("delete: %v\n", ds.String())
	ex = "[0 0 0 0 0 1 1 0 ]"
	out = ds.String()
	if out != ex {
		t.Errorf("Delete(2,4) != %v", out)
	}

	is := bs.Insert(3, 2)
	// fmt.Printf("insert: %v\n", is.String())
	ex = "[0 0 1 0 0 0 1 0 0 0 0 1 1 0 ]"
	out = is.String()
	if out != ex {
		t.Errorf("Insert(3,2) != %v", out)
	}
}

func TestBitSlice8(t *testing.T) {
	bs := Make(8, 0)

	ln := bs.Len()
	if ln != 8 {
		t.Errorf("len: %v != 8\n", ln)
	}

	// fmt.Printf("empty: %v\n", bs.String())
	var ex, out string
	ex = "[0 0 0 0 0 0 0 0 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("empty != %v", out)
	}

	bs.Append(true)
	ln = bs.Len()
	if ln != 9 {
		t.Errorf("len: %v != 9\n", ln)
	}
	// fmt.Printf("append true: %v\n", bs.String())
	ex = "[0 0 0 0 0 0 0 0 1 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("append true != %v", out)
	}

	bs.Append(false)
	ln = bs.Len()
	if ln != 10 {
		t.Errorf("len: %v != 10\n", ln)
	}
	// fmt.Printf("append false: %v\n", bs.String())
	ex = "[0 0 0 0 0 0 0 0 1 0 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("append false != %v", out)
	}
}

func TestBitSlice7(t *testing.T) {
	bs := Make(7, 0)

	ln := bs.Len()
	if ln != 7 {
		t.Errorf("len: %v != 7\n", ln)
	}

	// fmt.Printf("empty: %v\n", bs.String())
	var ex, out string
	ex = "[0 0 0 0 0 0 0 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("empty != %v", out)
	}

	bs.Append(true)
	ln = bs.Len()
	if ln != 8 {
		t.Errorf("len: %v != 8\n", ln)
	}
	// fmt.Printf("append true: %v\n", bs.String())
	ex = "[0 0 0 0 0 0 0 1 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("append true != %v", out)
	}

	bs.Append(false)
	ln = bs.Len()
	if ln != 9 {
		t.Errorf("len: %v != 9\n", ln)
	}
	// fmt.Printf("append false: %v\n", bs.String())
	ex = "[0 0 0 0 0 0 0 1 0 ]"
	out = bs.String()
	if out != ex {
		t.Errorf("append false != %v", out)
	}
}
