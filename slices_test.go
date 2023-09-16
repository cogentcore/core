// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

import (
	"testing"
)

type TstStruct struct {
	Field1 string
}

func TestCopySlice(t *testing.T) {
	var tof []float32
	var tos []string
	fmf := []float32{1, 2, 3}
	fms := []string{"3", "4", "5"}
	err := CopySliceRobust(&tof, fmf)
	if err != nil {
		t.Errorf("copy float: %s\n", err.Error())
	}
	for i := range fmf {
		if tof[i] != fmf[i] {
			t.Errorf("tof %d: %g != %g\n", i, tof[i], fmf[i])
		}
	}
	err = CopySliceRobust(&tos, fms)
	if err != nil {
		t.Errorf("copy string: %s\n", err.Error())
	}
	for i := range fms {
		if tos[i] != fms[i] {
			t.Errorf("tos %d: %s != %s\n", i, tos[i], fms[i])
		}
	}
	err = CopySliceRobust(&tof, fms)
	if err != nil {
		t.Errorf("copy float = string: %s\n", err.Error())
	}
	for i := range fms {
		if ToString(tof[i]) != fms[i] {
			t.Errorf("tof %d: %g != %s\n", i, tof[i], fms[i])
		}
	}
	fms = fms[:2]
	err = CopySliceRobust(&tof, fms)
	if err != nil {
		t.Errorf("copy float = string: %s\n", err.Error())
	}
	if len(tof) != len(fms) {
		t.Errorf("copy float = string: size not 2: %d\n", len(tof))
	}
	for i := range fms {
		if ToString(tof[i]) != fms[i] {
			t.Errorf("tof %d: %g != %s\n", i, tof[i], fms[i])
		}
	}
	fms = append(fms, "7")
	err = CopySliceRobust(&tof, fms)
	if err != nil {
		t.Errorf("copy float = string: %s\n", err.Error())
	}
	if len(tof) != len(fms) {
		t.Errorf("copy float = string: size not 3: %d\n", len(tof))
	}
	for i := range fms {
		if ToString(tof[i]) != fms[i] {
			t.Errorf("tof %d: %g != %s\n", i, tof[i], fms[i])
		}
	}

	var toc [][]float32
	fmc := [][]string{[]string{"1", "2"}, []string{"3", "4"}}
	err = CopySliceRobust(&toc, fmc)
	if err != nil {
		t.Errorf("copy [][]float = [][]string: %s\n", err.Error())
	}
	for i := range fmc {
		fmci := fmc[i]
		toci := toc[i]
		for j := range fmci {
			if ToString(toci[j]) != fmci[j] {
				t.Errorf("toci,j %d,%d: %g != %s\n", i, j, toci[j], fmci[j])
			}
		}
	}
}

// test slice type functions
func TestSliceType(t *testing.T) {
	var sl []string

	ts := SliceElType(sl).String()
	if ts != "string" {
		t.Errorf("slice el type should be string, not: %v\n", ts)
	}

	ts = SliceElType(&sl).String()
	if ts != "string" {
		t.Errorf("slice el type should be string, not: %v\n", ts)
	}

	var slp []*string

	ts = SliceElType(slp).String()
	if ts != "*string" {
		t.Errorf("slice el type should be *string, not: %v\n", ts)
	}

	ts = SliceElType(&slp).String()
	if ts != "*string" {
		t.Errorf("slice el type should be *string, not: %v\n", ts)
	}

	var slsl [][]string

	ts = SliceElType(slsl).String()
	if ts != "[]string" {
		t.Errorf("slice el type should be []string, not: %v\n", ts)
	}

	ts = SliceElType(&slsl).String()
	if ts != "[]string" {
		t.Errorf("slice el type should be []string, not: %v\n", ts)
	}

	// fmt.Printf("slsl kind: %v\n", SliceElType(slsl).Kind())

}
