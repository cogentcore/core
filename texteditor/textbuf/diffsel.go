// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"slices"

	"github.com/goki/go-difflib/difflib"
)

// DiffSelData contains data for one set of text
type DiffSelData struct {
	// original text
	Orig []string

	// edits applied
	Edit []string

	// mapping of original line numbers (index) to edited line numbers,
	// accounting for the edits applied so far
	LineMap []int

	// todo: in principle one should be able to reverse the edits to undo
	// but the orig is different -- figure it out later..

	// Undos: stack of diffs applied
	Undos Diffs

	// undo records
	EditUndo [][]string

	// undo records for ALineMap
	LineMapUndo [][]int
}

// SetStringLines sets the data from given lines of strings
// The Orig is set directly and Edit is cloned
// if the input will be modified during the processing,
// call slices.Clone first
func (ds *DiffSelData) SetStringLines(s []string) {
	ds.Orig = s
	ds.Edit = slices.Clone(s)
	nl := len(s)
	ds.LineMap = make([]int, nl)
	for i := range ds.LineMap {
		ds.LineMap[i] = i
	}
}

func (ds *DiffSelData) SaveUndo(op difflib.OpCode) {
	ds.Undos = append(ds.Undos, op)
	ds.EditUndo = append(ds.EditUndo, slices.Clone(ds.Edit))
	ds.LineMapUndo = append(ds.LineMapUndo, slices.Clone(ds.LineMap))
}

func (ds *DiffSelData) Undo() bool {
	n := len(ds.LineMapUndo)
	if n == 0 {
		return false
	}
	ds.Undos = ds.Undos[:n-1]
	ds.LineMap = ds.LineMapUndo[n-1]
	ds.LineMapUndo = ds.LineMapUndo[:n-1]
	ds.Edit = ds.EditUndo[n-1]
	ds.EditUndo = ds.EditUndo[:n-1]
	return true
}

// ApplyOneDiff applies given diff operator to given "B" lines
// using original "A" lines and given b line map
func ApplyOneDiff(op difflib.OpCode, bedit *[]string, aorig []string, blmap []int) {
	// fmt.Println("applying:", DiffOpString(op))
	switch op.Tag {
	case 'r':
		na := op.J2 - op.J1
		nb := op.I2 - op.I1
		b1 := blmap[op.I1]
		nc := min(na, nb)
		for i := 0; i < nc; i++ {
			(*bedit)[b1+i] = aorig[op.J1+i]
		}
		db := na - nb
		if db > 0 {
			*bedit = slices.Insert(*bedit, b1+nb, aorig[op.J1+nb:op.J2]...)
		} else {
			*bedit = slices.Delete(*bedit, b1+na, b1+nb)
		}
		for i := op.I2; i < len(blmap); i++ {
			blmap[i] = blmap[i] + db
		}
	case 'd':
		nb := op.I2 - op.I1
		b1 := blmap[op.I1]
		*bedit = slices.Delete(*bedit, b1, b1+nb)
		for i := op.I2; i < len(blmap); i++ {
			blmap[i] = blmap[i] - nb
		}
	case 'i':
		na := op.J2 - op.J1
		b1 := op.I1
		if op.I1 < len(blmap) {
			b1 = blmap[op.I1]
		} else {
			b1 = len(*bedit)
		}
		*bedit = slices.Insert(*bedit, b1, aorig[op.J1:op.J2]...)
		for i := op.I2; i < len(blmap); i++ {
			blmap[i] = blmap[i] + na
		}
	}
}

// DiffSelected supports the incremental application of selected diffs
// between two files (either A -> B or B <- A), with Undo
type DiffSelected struct {
	A DiffSelData
	B DiffSelData

	// Diffs are the diffs between A and B
	Diffs Diffs
}

func NewDiffSelected(astr, bstr []string) *DiffSelected {
	ds := &DiffSelected{}
	ds.SetStringLines(astr, bstr)
	return ds
}

// SetStringLines sets the data from given lines of strings
func (ds *DiffSelected) SetStringLines(astr, bstr []string) {
	ds.A.SetStringLines(astr)
	ds.B.SetStringLines(bstr)
	ds.Diffs = DiffLines(astr, bstr)
}

// AtoB applies given diff index from A to B
func (ds *DiffSelected) AtoB(idx int) {
	op := DiffOpReverse(ds.Diffs[idx])
	ds.B.SaveUndo(op)
	ApplyOneDiff(op, &ds.B.Edit, ds.A.Orig, ds.B.LineMap)
}

// BtoA applies given diff index from B to A
func (ds *DiffSelected) BtoA(idx int) {
	op := ds.Diffs[idx]
	ds.A.SaveUndo(op)
	ApplyOneDiff(op, &ds.A.Edit, ds.B.Orig, ds.A.LineMap)
}
