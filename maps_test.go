// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

import (
	"testing"
)

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

// test map type functions
func TestMapType(t *testing.T) {
	var mp map[string]int

	ts := MapValueType(mp).String()
	if ts != "int" {
		t.Errorf("map val type should be int, not: %v\n", ts)
	}

	ts = MapValueType(&mp).String()
	if ts != "int" {
		t.Errorf("map val type should be int, not: %v\n", ts)
	}

	ts = MapKeyType(mp).String()
	if ts != "string" {
		t.Errorf("map key type should be string, not: %v\n", ts)
	}

	ts = MapKeyType(&mp).String()
	if ts != "string" {
		t.Errorf("map key type should be string, not: %v\n", ts)
	}

}
