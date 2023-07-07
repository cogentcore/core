// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"fmt"
	"testing"
)

func AFun(aa any) bool {
	return IfaceIsNil(aa)
}

func TestIfaceIsNil(t *testing.T) {
	ai := any(a)

	if IfaceIsNil(ai) != false {
		t.Errorf("should be non-nil: %v\n", ai)
	}

	var ap *A
	api := any(ap)

	if IfaceIsNil(api) != true {
		t.Errorf("should be nil: %v\n", api)
	}

	if AFun(ap) != true {
		t.Errorf("should be nil: %v\n", ap)
	}

	if AFun(&a) != false {
		t.Errorf("should be non-nil: %v\n", &a)
	}

}

func TestConverts(t *testing.T) {
	fv := 3.14
	iv := 10
	sv := "25"
	// bv := true

	// note: this does not work
	// reflect.ValueOf(&fv).Elem().Set(reflect.ValueOf("1.58").Convert(reflect.TypeOf(fv)))
	ok := false

	ft := "1.58"
	ok = SetRobust(&fv, ft)
	fs := fmt.Sprintf("%v", fv)
	if !ok || fs != ft {
		t.Errorf("Float convert error: %v != %v, ok: %v\n", fs, ft, ok)
	}

	it := "1"
	ok = SetRobust(&iv, true)
	is := fmt.Sprintf("%v", iv)
	if !ok || is != it {
		t.Errorf("Int convert error: %v != %v, ok: %v\n", is, it, ok)
	}

	st := "22"
	ok = SetRobust(&sv, 22)
	ss := fmt.Sprintf("%v", sv)
	if !ok || ss != st {
		t.Errorf("String convert error: %v != %v, ok: %v\n", ss, st, ok)
	}
	tc := C{}
	InitC()
	ok = SetRobust(&tc, c)
	// fmt.Printf("tc %+v\n", tc)
	if !ok || tc != c {
		t.Errorf("Struct convert error: %+v != %+v, ok: %v\n", c, tc, ok)
	}
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

func TestCopyMap(t *testing.T) {
	var tof map[string]float32
	var tos map[string]string
	fmf := map[string]float32{"a": 1, "b": 2, "c": 3}
	fms := map[string]string{"a": "3", "b": "4", "c": "5"}
	err := CopyMapRobust(&tof, fmf)
	if err != nil {
		t.Errorf("copy float: %s\n", err.Error())
	}
	for i := range fmf {
		if tof[i] != fmf[i] {
			t.Errorf("tof %s: %g != %g\n", i, tof[i], fmf[i])
		}
	}
	err = CopyMapRobust(&tos, fms)
	if err != nil {
		t.Errorf("copy string: %s\n", err.Error())
	}
	for i := range fms {
		if tos[i] != fms[i] {
			t.Errorf("tos %s: %s != %s\n", i, tos[i], fms[i])
		}
	}
	err = CopyMapRobust(&tof, fms)
	if err != nil {
		t.Errorf("copy float = string: %s\n", err.Error())
	}
	for i := range fms {
		if ToString(tof[i]) != fms[i] {
			t.Errorf("tof %s: %g != %s\n", i, tof[i], fms[i])
		}
	}
	delete(fms, "b")
	err = CopyMapRobust(&tof, fms)
	if err != nil {
		t.Errorf("copy float = string: %s\n", err.Error())
	}
	if len(tof) != len(fms) {
		t.Errorf("copy float = string: size not 2: %d\n", len(tof))
	}
	for i := range fms {
		if ToString(tof[i]) != fms[i] {
			t.Errorf("tof %s: %g != %s\n", i, tof[i], fms[i])
		}
	}
	fms["e"] = "7"
	err = CopyMapRobust(&tof, fms)
	if err != nil {
		t.Errorf("copy float = string: %s\n", err.Error())
	}
	if len(tof) != len(fms) {
		t.Errorf("copy float = string: size not 3: %d\n", len(tof))
	}
	for i := range fms {
		if ToString(tof[i]) != fms[i] {
			t.Errorf("tof %s: %g != %s\n", i, tof[i], fms[i])
		}
	}

	var toc map[string]map[string]float32
	fmc := map[string]map[string]string{"q": {"a": "1", "b": "2"}, "z": {"c": "3", "d": "4"}}
	err = CopyMapRobust(&toc, fmc)
	if err != nil {
		t.Errorf("copy map[string]map[string]float = map[string]map[string]string: %s\n", err.Error())
	}
	for i := range fmc {
		fmci := fmc[i]
		toci := toc[i]
		for j := range fmci {
			if ToString(toci[j]) != fmci[j] {
				t.Errorf("toci,j %s,%s: %g != %s\n", i, j, toci[j], fmci[j])
			}
		}
	}
}
