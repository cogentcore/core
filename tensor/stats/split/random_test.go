// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package split

import (
	"fmt"
	"testing"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

func TestPermuted(t *testing.T) {
	dt := table.New(table.Schema{
		{"Name", tensor.STRING, nil, nil},
		{"Input", tensor.FLOAT32, []int{5, 5}, []string{"Y", "X"}},
		{"Output", tensor.FLOAT32, []int{5, 5}, []string{"Y", "X"}},
	}, 25)
	ix := table.NewIndexView(dt)
	spl, err := Permuted(ix, []float64{.5, .5}, nil)
	if err != nil {
		t.Error(err)
	}
	for i, sp := range spl.Splits {
		fmt.Printf("split: %v name: %v len: %v idxs: %v\n", i, spl.Values[i], len(sp.Indexes), sp.Indexes)
	}

	spl, err = Permuted(ix, []float64{.25, .5, .25}, []string{"test", "train", "validate"})
	if err != nil {
		t.Error(err)
	}
	for i, sp := range spl.Splits {
		fmt.Printf("split: %v name: %v len: %v idxs: %v\n", i, spl.Values[i], len(sp.Indexes), sp.Indexes)
	}
}
