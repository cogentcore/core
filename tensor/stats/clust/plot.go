// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clust

import (
	"cogentcore.org/core/tensor/stats/simat"
	"cogentcore.org/core/tensor/table"
)

// Plot sets the rows of given data table to trace out lines with labels that
// will render cluster plot starting at root node when plotted with a standard plotting package.
// The lines double-back on themselves to form a continuous line to be plotted.
func Plot(pt *table.Table, root *Node, smat *simat.SimMat) {
	pt.DeleteAll()
	pt.AddFloat64Column("X")
	pt.AddFloat64Column("Y")
	pt.AddStringColumn("Label")
	nextY := 0.5
	root.SetYs(&nextY)
	root.SetParDist(0.0)
	root.Plot(pt, smat)
}

// Plot sets the rows of given data table to trace out lines with labels that
// will render this node in a cluster plot when plotted with a standard plotting package.
// The lines double-back on themselves to form a continuous line to be plotted.
func (nn *Node) Plot(pt *table.Table, smat *simat.SimMat) {
	row := pt.Rows
	if nn.IsLeaf() {
		pt.SetNumRows(row + 1)
		pt.SetFloatIndex(0, row, nn.ParDist)
		pt.SetFloatIndex(1, row, nn.Y)
		if len(smat.Rows) > nn.Index {
			pt.SetStringIndex(2, row, smat.Rows[nn.Index])
		}
	} else {
		for _, kn := range nn.Kids {
			pt.SetNumRows(row + 2)
			pt.SetFloatIndex(0, row, nn.ParDist)
			pt.SetFloatIndex(1, row, kn.Y)
			row++
			pt.SetFloatIndex(0, row, nn.ParDist+nn.Dist)
			pt.SetFloatIndex(1, row, kn.Y)
			kn.Plot(pt, smat)
			row = pt.Rows
			pt.SetNumRows(row + 1)
			pt.SetFloatIndex(0, row, nn.ParDist)
			pt.SetFloatIndex(1, row, kn.Y)
			row++
		}
		pt.SetNumRows(row + 1)
		pt.SetFloatIndex(0, row, nn.ParDist)
		pt.SetFloatIndex(1, row, nn.Y)
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
