// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensormpi

import (
	"cogentcore.org/core/base/mpi"
	"cogentcore.org/core/tensor/table"
)

// GatherTableRows does an MPI AllGather on given src table data, gathering into dest.
// dest will have np * src.Rows Rows, filled with each processor's data, in order.
// dest must be a clone of src: if not same number of cols, will be configured from src.
func GatherTableRows(dest, src *table.Table, comm *mpi.Comm) {
	sr := src.Rows
	np := mpi.WorldSize()
	dr := np * sr
	if len(dest.Columns) != len(src.Columns) {
		*dest = *src.Clone()
	}
	dest.SetNumRows(dr)
	for ci, st := range src.Columns {
		dt := dest.Columns[ci]
		GatherTensorRows(dt, st, comm)
	}
}

// ReduceTable does an MPI AllReduce on given src table data using given operation,
// gathering into dest.
// each processor must have the same table organization -- the tensor values are
// just aggregated directly across processors.
// dest will be a clone of src if not the same (cos & rows),
// does nothing for strings.
func ReduceTable(dest, src *table.Table, comm *mpi.Comm, op mpi.Op) {
	sr := src.Rows
	if len(dest.Columns) != len(src.Columns) {
		*dest = *src.Clone()
	}
	dest.SetNumRows(sr)
	for ci, st := range src.Columns {
		dt := dest.Columns[ci]
		ReduceTensor(dt, st, comm, op)
	}
}
