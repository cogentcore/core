// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensormpi

import (
	"reflect"

	"cogentcore.org/core/base/mpi"
	"cogentcore.org/core/tensor"
)

// GatherTensorRows does an MPI AllGather on given src tensor data, gathering into dest,
// using a row-based tensor organization (as in an table.Table).
// dest will have np * src.Rows Rows, filled with each processor's data, in order.
// dest must have same overall shape as src at start, but rows will be enforced.
func GatherTensorRows(dest, src tensor.Tensor, comm *mpi.Comm) error {
	dt := src.DataType()
	if dt == reflect.String {
		return GatherTensorRowsString(dest.(*tensor.String), src.(*tensor.String), comm)
	}
	sr, _ := src.RowCellSize()
	dr, _ := dest.RowCellSize()
	np := mpi.WorldSize()
	dl := np * sr
	if dr != dl {
		dest.SetNumRows(dl)
		dr = dl
	}

	var err error
	switch dt {
	case reflect.Bool:
		// todo
	case reflect.Uint8:
		dt := dest.(*tensor.Byte)
		st := src.(*tensor.Byte)
		err = comm.AllGatherU8(dt.Values, st.Values)
	case reflect.Int32:
		dt := dest.(*tensor.Int32)
		st := src.(*tensor.Int32)
		err = comm.AllGatherI32(dt.Values, st.Values)
	case reflect.Int:
		dt := dest.(*tensor.Int)
		st := src.(*tensor.Int)
		err = comm.AllGatherInt(dt.Values, st.Values)
	case reflect.Float32:
		dt := dest.(*tensor.Float32)
		st := src.(*tensor.Float32)
		err = comm.AllGatherF32(dt.Values, st.Values)
	case reflect.Float64:
		dt := dest.(*tensor.Float64)
		st := src.(*tensor.Float64)
		err = comm.AllGatherF64(dt.Values, st.Values)
	}
	return err
}

// GatherTensorRowsString does an MPI AllGather on given String src tensor data,
// gathering into dest, using a row-based tensor organization (as in an table.Table).
// dest will have np * src.Rows Rows, filled with each processor's data, in order.
// dest must have same overall shape as src at start, but rows will be enforced.
func GatherTensorRowsString(dest, src *tensor.String, comm *mpi.Comm) error {
	sr, _ := src.RowCellSize()
	dr, _ := dest.RowCellSize()
	np := mpi.WorldSize()
	dl := np * sr
	if dr != dl {
		dest.SetNumRows(dl)
		dr = dl
	}
	ssz := len(src.Values)
	dsz := len(dest.Values)
	sln := make([]int, ssz)
	dln := make([]int, dsz)
	for i, s := range src.Values {
		sln[i] = len(s)
	}
	err := comm.AllGatherInt(dln, sln)
	if err != nil {
		return err
	}
	mxlen := 0
	for _, l := range dln {
		mxlen = max(mxlen, l)
	}
	if mxlen == 0 {
		return nil // nothing to transfer
	}
	sdt := make([]byte, ssz*mxlen)
	ddt := make([]byte, dsz*mxlen)
	idx := 0
	for _, s := range src.Values {
		l := len(s)
		copy(sdt[idx:idx+l], []byte(s))
		idx += mxlen
	}
	err = comm.AllGatherU8(ddt, sdt)
	idx = 0
	for i := range dest.Values {
		l := dln[i]
		s := string(ddt[idx : idx+l])
		dest.Values[i] = s
		idx += mxlen
	}
	return err
}

// ReduceTensor does an MPI AllReduce on given src tensor data, using given operation,
// gathering into dest.  dest must have same overall shape as src -- will be enforced.
// IMPORTANT: src and dest must be different slices!
// each processor must have the same shape and organization for this to make sense.
// does nothing for strings.
func ReduceTensor(dest, src tensor.Tensor, comm *mpi.Comm, op mpi.Op) error {
	dt := src.DataType()
	if dt == reflect.String {
		return nil
	}
	slen := src.Len()
	if slen != dest.Len() {
		dest.CopyShapeFrom(src)
	}
	var err error
	switch dt {
	case reflect.Bool:
		dt := dest.(*tensor.Bits)
		st := src.(*tensor.Bits)
		err = comm.AllReduceU8(op, dt.Values, st.Values)
	case reflect.Uint8:
		dt := dest.(*tensor.Byte)
		st := src.(*tensor.Byte)
		err = comm.AllReduceU8(op, dt.Values, st.Values)
	case reflect.Int32:
		dt := dest.(*tensor.Int32)
		st := src.(*tensor.Int32)
		err = comm.AllReduceI32(op, dt.Values, st.Values)
	case reflect.Int:
		dt := dest.(*tensor.Int)
		st := src.(*tensor.Int)
		err = comm.AllReduceInt(op, dt.Values, st.Values)
	case reflect.Float32:
		dt := dest.(*tensor.Float32)
		st := src.(*tensor.Float32)
		err = comm.AllReduceF32(op, dt.Values, st.Values)
	case reflect.Float64:
		dt := dest.(*tensor.Float64)
		st := src.(*tensor.Float64)
		err = comm.AllReduceF64(op, dt.Values, st.Values)
	}
	return err
}
