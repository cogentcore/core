// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cluster

import (
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

// Plot sets the rows of given data table to trace out lines with labels that
// will render cluster plot starting at root node when plotted with a standard plotting package.
// The lines double-back on themselves to form a continuous line to be plotted.
func Plot(pt *table.Table, root *Node, dmat, labels tensor.Tensor) {
	pt.DeleteAll()
	pt.AddFloat64Column("X")
	pt.AddFloat64Column("Y")
	pt.AddStringColumn("Label")
	nextY := 0.5
	root.SetYs(&nextY)
	root.SetParDist(0.0)
	root.Plot(pt, dmat, labels)
}

// Plot sets the rows of given data table to trace out lines with labels that
// will render this node in a cluster plot when plotted with a standard plotting package.
// The lines double-back on themselves to form a continuous line to be plotted.
func (nn *Node) Plot(pt *table.Table, dmat, labels tensor.Tensor) {
	row := pt.NumRows()
	xc := pt.ColumnByIndex(0)
	yc := pt.ColumnByIndex(1)
	lbl := pt.ColumnByIndex(2)
	if nn.IsLeaf() {
		pt.SetNumRows(row + 1)
		xc.SetFloatRow(nn.ParDist, row, 0)
		yc.SetFloatRow(nn.Y, row, 0)
		if labels.Len() > nn.Index {
			lbl.SetStringRow(labels.StringValue(nn.Index), row, 0)
		}
	} else {
		for _, kn := range nn.Kids {
			pt.SetNumRows(row + 2)
			xc.SetFloatRow(nn.ParDist, row, 0)
			yc.SetFloatRow(kn.Y, row, 0)
			row++
			xc.SetFloatRow(nn.ParDist+nn.Dist, row, 0)
			yc.SetFloatRow(kn.Y, row, 0)
			kn.Plot(pt, dmat, labels)
			row = pt.NumRows()
			pt.SetNumRows(row + 1)
			xc.SetFloatRow(nn.ParDist, row, 0)
			yc.SetFloatRow(kn.Y, row, 0)
			row++
		}
		pt.SetNumRows(row + 1)
		xc.SetFloatRow(nn.ParDist, row, 0)
		yc.SetFloatRow(nn.Y, row, 0)
	}
}

// SetYs sets the Y-axis values for the nodes in preparation for plotting.
func (nn *Node) SetYs(nextY *float64) {
	if nn.IsLeaf() {
		nn.Y = *nextY
		(*nextY) += 1.0
	} else {
		avgy := 0.0
		for _, kn := range nn.Kids {
			kn.SetYs(nextY)
			avgy += kn.Y
		}
		avgy /= float64(len(nn.Kids))
		nn.Y = avgy
	}
}

// SetParDist sets the parent distance for the nodes in preparation for plotting.
func (nn *Node) SetParDist(pard float64) {
	nn.ParDist = pard
	if !nn.IsLeaf() {
		pard += nn.Dist
		for _, kn := range nn.Kids {
			kn.SetParDist(pard)
		}
	}
}
