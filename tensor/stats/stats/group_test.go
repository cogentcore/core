// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"testing"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorfs"
	"github.com/stretchr/testify/assert"
)

func TestGroup(t *testing.T) {
	dt := table.New().SetNumRows(4)
	dt.AddStringColumn("Name")
	dt.AddFloat32Column("Value")
	for i := range dt.NumRows() {
		gp := "A"
		if i >= 2 {
			gp = "B"
		}
		dt.Column("Name").SetStringRow(gp, i, 0)
		dt.Column("Value").SetFloatRow(float64(i), i, 0)
	}
	dir, _ := tensorfs.NewDir("Group")
	err := TableGroups(dir, dt, "Name")
	assert.NoError(t, err)

	ixs := dir.ValuesFunc(nil)
	assert.Equal(t, []int{0, 1}, tensor.AsInt(ixs[0]).Values)
	assert.Equal(t, []int{2, 3}, tensor.AsInt(ixs[1]).Values)

	err = TableGroupStats(dir, StatMean, dt, "Value")
	assert.NoError(t, err)

	gdt := GroupStatsAsTableNoStatName(dir)
	assert.Equal(t, 0.5, gdt.Column("Value").Float1D(0))
	assert.Equal(t, 2.5, gdt.Column("Value").Float1D(1))
	assert.Equal(t, "A", gdt.Column("Name").String1D(0))
	assert.Equal(t, "B", gdt.Column("Name").String1D(1))
}

/*
func TestAggEmpty(t *testing.T) {
	dt := table.New().SetNumRows(4)
	dt.AddStringColumn("Group")
	dt.AddFloat32Column("Value")
	for i := 0; i < dt.Rows; i++ {
		gp := "A"
		if i >= 2 {
			gp = "B"
		}
		dt.SetString("Group", i, gp)
		dt.SetFloat("Value", i, float64(i))
	}
	ix := table.NewIndexed(dt)
	ix.Filter(func(et *table.Table, row int) bool {
		return false // exclude all
	})
	spl := GroupBy(ix, "Group")
	assert.Equal(t, 1, len(spl.Splits))

	AggColumn(spl, "Value", stats.Mean)

	st := spl.AggsToTable(table.ColumnNameOnly)
	if st == nil {
		t.Error("AggsToTable should not be nil!")
	}
}
*/
