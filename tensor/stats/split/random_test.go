// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package split

import (
	"testing"

	"cogentcore.org/core/tensor/table"

	"github.com/stretchr/testify/assert"
)

func TestPermuted(t *testing.T) {
	dt := table.NewTable().SetNumRows(25)
	dt.AddStringColumn("Name")
	dt.AddFloat32TensorColumn("Input", []int{5, 5}, "Y", "X")
	dt.AddFloat32TensorColumn("Output", []int{5, 5}, "Y", "X")
	ix := table.NewIndexView(dt)
	spl, err := Permuted(ix, []float64{.5, .5}, nil)
	if err != nil {
		t.Error(err)
	}
	// for i, sp := range spl.Splits {
	// 	fmt.Printf("split: %v name: %v len: %v idxs: %v\n", i, spl.Values[i], len(sp.Indexes), sp.Indexes)
	// }
	assert.Equal(t, 2, len(spl.Splits))
	assert.Contains(t, []int{12, 13}, len(spl.Splits[0].Indexes))
	assert.Contains(t, []int{12, 13}, len(spl.Splits[1].Indexes))

	spl, err = Permuted(ix, []float64{.25, .5, .25}, []string{"test", "train", "validate"})
	if err != nil {
		t.Error(err)
	}
	// for i, sp := range spl.Splits {
	// 	fmt.Printf("split: %v name: %v len: %v idxs: %v\n", i, spl.Values[i], len(sp.Indexes), sp.Indexes)
	// }
	assert.Equal(t, 3, len(spl.Splits))
	assert.Equal(t, 6, len(spl.Splits[0].Indexes))
	assert.Equal(t, 13, len(spl.Splits[1].Indexes))
	assert.Equal(t, 6, len(spl.Splits[2].Indexes))
}
